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
//
// Robustesse (fix 17/10) : chaque client a une goroutine d'écriture dédiée
// (writeLoop) alimentée par un outbox bufferisé. Le mutex global n'est JAMAIS
// bloqué par une écriture TCP lente — un client qui ne consomme pas ses
// snapshots est simplement déconnecté (outbox plein), sans impacter les
// autres joueurs.
package server

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"time"

	"game-test/internal/game"
	netproto "game-test/internal/net"
)

// outboxSize est la capacité du buffer d'écriture par client. Au-delà, le
// client est considéré comme trop lent et est déconnecté.
const outboxSize = 256

// writeTimeout est le délai max d'une écriture TCP (évite un blocage infini
// si le client ne lit plus du tout).
const writeTimeout = 10 * time.Second

// Server est le serveur de jeu TCP. Il détient l'état autoritaire de la
// partie et accepte les connexions des clients (multi : plusieurs joueurs
// partagent le même monde).
type Server struct {
	state   *game.State
	ln      net.Listener
	mu      sync.Mutex           // protège state, players, clients
	players map[int]*game.Player // playerID → joueur
	clients map[int]*clientConn  // playerID → connexion (outbox + writeLoop)
	nextID  int                  // prochain ID de joueur
}

// clientConn est la connexion d'un client côté serveur. Une goroutine
// writeLoop écrit les messages de l'outbox sur le TCP : le mutex global
// n'est jamais bloqué par une écriture lente.
type clientConn struct {
	conn   net.Conn
	outbox chan []byte
	done   chan struct{}
	once   sync.Once
}

func newClientConn(conn net.Conn) *clientConn {
	cc := &clientConn{
		conn:   conn,
		outbox: make(chan []byte, outboxSize),
		done:   make(chan struct{}),
	}
	go cc.writeLoop()
	return cc
}

// writeLoop écrit les messages de l'outbox sur la connexion TCP.
func (cc *clientConn) writeLoop() {
	for {
		select {
		case data := <-cc.outbox:
			cc.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if _, err := cc.conn.Write(data); err != nil {
				return // déconnexion
			}
		case <-cc.done:
			return
		}
	}
}

// send pousse un message dans l'outbox (non bloquant). Retourne false si
// l'outbox est plein (client trop lent) ou fermé.
func (cc *clientConn) send(data []byte) bool {
	if data == nil {
		return true
	}
	select {
	case cc.outbox <- data:
		return true
	case <-cc.done:
		return false
	default:
		return false // outbox plein : client trop lent
	}
}

// close ferme la connexion (idempotent).
func (cc *clientConn) close() {
	cc.once.Do(func() {
		close(cc.done)
		cc.conn.Close()
	})
}

// NewServer crée un serveur avec une nouvelle partie (seed, dimensions).
func NewServer(seed int64, width, height int) *Server {
	return NewServerMode(seed, width, height, game.ModeCoop)
}

// NewServerMode crée un serveur avec le mode de jeu donné (COOP ou MMORPG).
func NewServerMode(seed int64, width, height int, mode game.GameMode) *Server {
	var st *game.State
	if mode == game.ModeMMORPG {
		st = game.NewStateMMORPG(seed, width, height)
	} else {
		st = game.NewStateSized(seed, width, height)
	}
	return &Server{
		state:   st,
		players: make(map[int]*game.Player),
		clients: make(map[int]*clientConn),
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

// pendingSend associe un clientConn à un message à lui envoyer.
type pendingSend struct {
	cc   *clientConn
	data []byte
}

// handleConn sert un client : crée un joueur, envoie Welcome + snapshot
// initial, puis boucle sur les inputs (applique → broadcast à tous).
func (s *Server) handleConn(conn net.Conn) {
	cc := newClientConn(conn)
	defer cc.close()

	// Créer un nouveau joueur pour ce client.
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	p := s.spawnPlayerLocked(id)
	s.players[id] = p
	s.clients[id] = cc
	// FOV initial créé par spawnPlayerLocked sur la map racine.
	rootMap := s.state.Maps.Get(p.MapID)
	var width, height int
	if rootMap != nil {
		width, height = rootMap.World.Width, rootMap.World.Height
	}
	var fov *game.FOV
	if rootMap != nil {
		fov = rootMap.FOVFor(id)
	}
	snap := s.state.SnapshotForPlayer(p, fov, id)
	welcome := encodeWelcomeMessage(s.state.Seed, width, height, id)
	snapData := encodeSnapshotMessage(snap)
	s.mu.Unlock()

	// Welcome + snapshot initial via l'outbox (écriture asynchrone).
	cc.send(welcome)
	cc.send(snapData)

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			s.removeClient(id)
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

		// Appliquer l'input sous mutex, préparer le broadcast.
		s.mu.Lock()
		// Swap : le joueur courant devient celui de ce client.
		s.state.Player = *s.players[id]
		// Others = joueurs sur la MÊME map (les monstres ne voient que la map courante).
		s.state.Others = s.otherPlayersOnMapLocked(id)
		s.applyInput(in)
		*s.players[id] = s.state.Player
		s.recomputeFOVsLocked()
		pending := s.buildBroadcastLocked()
		phase := s.state.Phase
		s.mu.Unlock()

		// Broadcast HORS mutex : push dans les outbox (non bloquant).
		// Un client trop lent (outbox plein) est déconnecté sans bloquer
		// les autres.
		for _, ps := range pending {
			if !ps.cc.send(ps.data) {
				go s.disconnectClient(ps.cc)
			}
		}
		if phase != game.PhasePlaying {
			cc.send(encodeEndMessage("partie terminée"))
			s.removeClient(id)
			return
		}
	}
}

// removeClient retire un client et son joueur (sous mutex).
func (s *Server) removeClient(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cc, ok := s.clients[id]; ok {
		cc.close()
		delete(s.clients, id)
	}
	delete(s.players, id)
}

// disconnectClient ferme un client (outbox plein = trop lent) et le retire.
func (s *Server) disconnectClient(cc *clientConn) {
	s.mu.Lock()
	for id, c := range s.clients {
		if c == cc {
			cc.close()
			delete(s.clients, id)
			delete(s.players, id)
			break
		}
	}
	s.mu.Unlock()
}

// spawnPlayerLocked crée un nouveau joueur sur la map racine (étage 1 en COOP).
// Le RNG est dérivé (seed + mapID + id) pour un spawn déterministe par joueur.
func (s *Server) spawnPlayerLocked(id int) *game.Player {
	// Map racine (étage 1 en COOP, overworld en MMORPG). Fallback : première map.
	rootMap := s.state.Maps.Root()
	if rootMap == nil {
		for _, m := range s.state.Maps.All() {
			rootMap = m
			break
		}
	}
	mapID := 0
	var x, y int
	if rootMap != nil {
		mapID = rootMap.ID
		rng := game.NewRNG(s.state.Seed*1000 + int64(rootMap.ID)*100 + int64(id))
		x, y, _ = rootMap.World.RandomFloor(rng)
		// Cherche une case libre si occupée.
		if !rootMap.World.IsWalkable(x, y) || s.playerOccupiedOnMapLocked(rootMap.ID, x, y, id) {
			for yy := 1; yy < rootMap.World.Height-1; yy++ {
				for xx := 1; xx < rootMap.World.Width-1; xx++ {
					if rootMap.World.IsWalkable(xx, yy) && !s.playerOccupiedOnMapLocked(rootMap.ID, xx, yy, id) {
						x, y = xx, yy
						break
					}
				}
			}
		}
	}
	if x == 0 && y == 0 {
		x, y = 1, 1
	}
	p := &game.Player{
		Entity:  *game.NewPlayer(x, y),
		ID:      id,
		MapID:   mapID,
		Level:   1,
		MaxMana: game.StartMaxMana,
		Mana:    game.StartMaxMana,
	}
	// FOV initial sur la map racine.
	if rootMap != nil {
		rootMap.SetFOV(id, game.NewFOV(rootMap.World.Width*rootMap.World.Height))
		game.ComputeFOV(rootMap.World, rootMap.FOVFor(id), x, y, game.PlayerViewRange)
	}
	return p
}

// playerOccupiedOnMapLocked indique si un autre joueur occupe (x, y) sur la map.
func (s *Server) playerOccupiedOnMapLocked(mapID, x, y, exceptID int) bool {
	for pid, op := range s.players {
		if pid != exceptID && op.MapID == mapID && op.X == x && op.Y == y {
			return true
		}
	}
	return false
}

// otherPlayersOnMapLocked retourne les autres joueurs sur la MÊME map que le
// joueur donné (pour state.Others pendant le tour d'un client).
func (s *Server) otherPlayersOnMapLocked(id int) []*game.Player {
	p := s.players[id]
	if p == nil {
		return nil
	}
	var others []*game.Player
	for pid, op := range s.players {
		if pid != id && op.MapID == p.MapID {
			others = append(others, op)
		}
	}
	return others
}

// recomputeFOVsLocked recalcule le FOV de chaque joueur dans SA map.
func (s *Server) recomputeFOVsLocked() {
	for id, p := range s.players {
		m := s.state.Maps.Get(p.MapID)
		if m == nil {
			continue
		}
		if m.FOVFor(id) == nil {
			m.SetFOV(id, game.NewFOV(m.World.Width*m.World.Height))
		}
		game.ComputeFOV(m.World, m.FOVFor(id), p.X, p.Y, game.PlayerViewRange)
	}
}

// buildBroadcastLocked construit les messages de snapshot pour chaque client
// connecté (sous mutex). Chaque joueur reçoit le snapshot de SA map avec SON
// FOV. L'envoi effectif se fait hors mutex via les outbox.
func (s *Server) buildBroadcastLocked() []pendingSend {
	out := make([]pendingSend, 0, len(s.players))
	for id, p := range s.players {
		m := s.state.Maps.Get(p.MapID)
		if m == nil {
			continue
		}
		fov := m.FOVFor(id)
		cc, ok := s.clients[id]
		if !ok {
			continue
		}
		snap := s.state.SnapshotForPlayer(p, fov, id)
		out = append(out, pendingSend{cc: cc, data: encodeSnapshotMessage(snap)})
	}
	return out
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
	case netproto.InputEnterBuilding:
		st.EnterBuilding()
	case netproto.InputExitBuilding:
		st.ExitBuilding()
	}
}

// encodeSnapshotMessage sérialise un snapshot en message JSON + newline.
func encodeSnapshotMessage(snap *game.Snapshot) []byte {
	env, err := netproto.Encode(netproto.MsgSnapshot, netproto.SnapshotPayload{Snap: snap})
	if err != nil {
		return nil
	}
	data, err := json.Marshal(env)
	if err != nil {
		return nil
	}
	return append(data, '\n')
}

// encodeWelcomeMessage sérialise un Welcome en JSON + newline.
func encodeWelcomeMessage(seed int64, width, height, playerID int) []byte {
	env, err := netproto.Encode(netproto.MsgWelcome, netproto.WelcomePayload{
		PlayerID: playerID,
		Seed:     seed,
		Width:    width,
		Height:   height,
	})
	if err != nil {
		return nil
	}
	data, err := json.Marshal(env)
	if err != nil {
		return nil
	}
	return append(data, '\n')
}

// encodeEndMessage sérialise un End en JSON + newline.
func encodeEndMessage(reason string) []byte {
	env, _ := netproto.Encode(netproto.MsgEnd, netproto.EndPayload{Reason: reason})
	data, _ := json.Marshal(env)
	return append(data, '\n')
}

// Close ferme le listener.
func (s *Server) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}
