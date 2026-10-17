// Package server implémente le serveur de jeu du Donjon Caféiné.
// Le serveur détient le game.State (autorité du gameplay), reçoit les inputs
// des clients, les applique, et publie des snapshots personnalisés (un par
// joueur, avec son FOV et sa position).
//
// Transport : TCP (cmd/coffee-dungeon-server). En solo, le même protocole
// tourne en mémoire via client.LocalClient (channels).
//
// Multijoueur (bloc 6) : plusieurs clients partagent le même monde. Chaque
// joueur a son propre FOV ; les monstres ciblent le joueur le plus proche
// (focus, géré dans game.focusPlayer).
package server

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"

	"game-test/internal/game"
	netproto "game-test/internal/net"
)

// Server est le serveur de jeu TCP. Il détient l'état autoritaire de la
// partie et accepte les connexions des clients (multi : plusieurs joueurs
// partagent le même monde).
type Server struct {
	state   *game.State
	ln      net.Listener
	mu      sync.Mutex           // protège state, players, fovs, conns
	players map[int]*game.Player // playerID → joueur
	fovs    map[int]*game.FOV    // playerID → FOV personnel
	conns   map[int]net.Conn     // playerID → connexion TCP
	nextID  int                  // prochain ID de joueur
}

// NewServer crée un serveur avec une nouvelle partie (seed, dimensions).
func NewServer(seed int64, width, height int) *Server {
	return &Server{
		state:   game.NewStateSized(seed, width, height),
		players: make(map[int]*game.Player),
		fovs:    make(map[int]*game.FOV),
		conns:   make(map[int]net.Conn),
	}
}

// Listen ouvre l'écoute TCP sur l'adresse donnée (ex: ":27015").
func (s *Server) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln
	return nil
}

// Addr renvoie l'adresse d'écoute (utile quand le port est 0 = auto).
func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

// Serve accepte les connexions en boucle. Chaque connexion est servie dans
// une goroutine dédiée.
func (s *Server) Serve() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// handleConn sert un client : crée un joueur, envoie Welcome + snapshot
// initial, puis boucle sur les inputs (applique → broadcast à tous).
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	// Créer un nouveau joueur pour ce client.
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	p := s.spawnPlayerLocked(id)
	s.players[id] = p
	s.fovs[id] = game.NewFOV(s.state.World.Width * s.state.World.Height)
	s.conns[id] = conn
	// FOV initial du nouveau joueur.
	game.ComputeFOV(s.state.World, s.fovs[id], p.X, p.Y, game.PlayerViewRange)
	snap := s.state.SnapshotForPlayer(p, s.fovs[id], id)
	s.mu.Unlock()

	// Welcome (playerID) puis snapshot initial personnalisé.
	s.sendWelcome(conn, id)
	s.sendSnapshotToLocked(conn, snap)

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// Déconnexion : retirer le joueur.
			s.mu.Lock()
			delete(s.players, id)
			delete(s.fovs, id)
			delete(s.conns, id)
			s.mu.Unlock()
			return
		}
		var env netproto.Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}
		if env.Type != netproto.MsgInput {
			continue
		}
		var in netproto.InputPayload
		if err := netproto.DecodePayload(env, &in); err != nil {
			continue
		}
		s.mu.Lock()
		// Swap : le joueur courant devient celui de ce client.
		s.state.Player = *s.players[id]
		s.state.Others = s.otherPlayersLocked(id)
		// Appliquer l'input (calcule le tour des monstres, qui voient tous
		// les joueurs via focusPlayer).
		s.applyInput(in)
		// Sauvegarder le joueur modifié.
		*s.players[id] = s.state.Player
		// Recalculer les FOV de tous les joueurs et broadcaster.
		s.recomputeFOVsLocked()
		s.broadcastSnapshotsLocked()
		phase := s.state.Phase
		s.mu.Unlock()
		if phase != game.PhasePlaying {
			s.sendEnd(conn, "partie terminée")
			return
		}
	}
}

// spawnPlayerLocked crée un nouveau joueur sur une case libre de l'étage.
// Le RNG est dérivé (seed + depth + id) pour un spawn déterministe par joueur.
func (s *Server) spawnPlayerLocked(id int) *game.Player {
	rng := game.NewRNG(s.state.Seed*1000 + int64(s.state.World.Depth)*100 + int64(id))
	x, y, ok := s.state.World.RandomFloor(rng)
	if !ok || s.state.Occupied(x, y) {
		// Cherche une case libre manuellement.
		for yy := 1; yy < s.state.World.Height-1 && !ok; yy++ {
			for xx := 1; xx < s.state.World.Width-1 && !ok; xx++ {
				if s.state.World.IsWalkable(xx, yy) && !s.state.Occupied(xx, yy) {
					x, y, ok = xx, yy, true
				}
			}
		}
	}
	if !ok {
		x, y = 1, 1
	}
	p := &game.Player{
		Entity:  *game.NewPlayer(x, y),
		ID:      id,
		Level:   1,
		MaxMana: game.StartMaxMana,
		Mana:    game.StartMaxMana,
	}
	return p
}

// otherPlayersLocked retourne les joueurs autres que celui donné (pour
// state.Others pendant le tour d'un client).
func (s *Server) otherPlayersLocked(id int) []*game.Player {
	others := make([]*game.Player, 0, len(s.players)-1)
	for pid, p := range s.players {
		if pid != id {
			others = append(others, p)
		}
	}
	return others
}

// recomputeFOVsLocked recalcule le FOV de chaque joueur connecté.
func (s *Server) recomputeFOVsLocked() {
	for id, p := range s.players {
		if s.fovs[id] == nil {
			s.fovs[id] = game.NewFOV(s.state.World.Width * s.state.World.Height)
		}
		game.ComputeFOV(s.state.World, s.fovs[id], p.X, p.Y, game.PlayerViewRange)
	}
}

// broadcastSnapshotsLocked envoie un snapshot personnalisé à chaque client
// (avec son propre joueur et son FOV).
func (s *Server) broadcastSnapshotsLocked() {
	for id, conn := range s.conns {
		p := s.players[id]
		fov := s.fovs[id]
		if p == nil || fov == nil {
			continue
		}
		snap := s.state.SnapshotForPlayer(p, fov, id)
		s.sendSnapshotToLocked(conn, snap)
	}
}

// applyInput applique une action du joueur sur le state (autorité serveur).
func (s *Server) applyInput(in netproto.InputPayload) {
	st := s.state
	switch in.Kind {
	case netproto.InputMove:
		st.MovePlayer(in.DX, in.DY)
	case netproto.InputWait:
		st.PlayerWait()
	case netproto.InputFireball:
		st.CastFireball(in.LastDX, in.LastDY)
	case netproto.InputDescend:
		st.Descend()
	case netproto.InputAscend:
		st.Ascend()
	case netproto.InputEquip:
		st.EquipItem(in.Index)
	case netproto.InputUse:
		st.UseItem(in.Index)
	case netproto.InputDrop:
		st.DropItem(in.Index)
	case netproto.InputBuy:
		st.BuyItem(in.Index)
	case netproto.InputSell:
		st.SellItem(in.Index)
	}
}

// sendWelcome envoie le message d'accueil (playerID, seed, dimensions).
func (s *Server) sendWelcome(conn net.Conn, playerID int) {
	env, err := netproto.Encode(netproto.MsgWelcome, netproto.WelcomePayload{
		PlayerID: playerID,
		Seed:     s.state.Seed,
		Width:    s.state.World.Width,
		Height:   s.state.World.Height,
	})
	if err != nil {
		return
	}
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	conn.Write(append(data, '\n'))
}

// sendSnapshotToLocked envoie un snapshot au client (JSON + newline).
// Le mutex doit être tenu par l'appelant.
func (s *Server) sendSnapshotToLocked(conn net.Conn, snap *game.Snapshot) {
	env, err := netproto.Encode(netproto.MsgSnapshot, netproto.SnapshotPayload{Snap: snap})
	if err != nil {
		return
	}
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	conn.Write(append(data, '\n'))
}

// sendEnd envoie un message de fin au client.
func (s *Server) sendEnd(conn net.Conn, reason string) {
	env, _ := netproto.Encode(netproto.MsgEnd, netproto.EndPayload{Reason: reason})
	data, _ := json.Marshal(env)
	conn.Write(append(data, '\n'))
}

// Close ferme le listener.
func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}
