// Package server implémente le serveur de jeu du Donjon Caféiné.
// Le serveur détient le game.State (autorité du gameplay), reçoit les inputs
// des clients, les applique, et publie des snapshots.
//
// Transport : TCP (cmd/coffee-dungeon-server). En solo, le même protocole
// tourne en mémoire via client.LocalClient (channels).
package server

import (
	"bufio"
	"encoding/json"
	"net"

	"game-test/internal/game"
	netproto "game-test/internal/net"
)

// Server est le serveur de jeu TCP. Il détient l'état autoritaire de la
// partie et accepte les connexions des clients.
//
// Bloc 5 : mono-partie (un state partagé). Le bloc 6 (multijoueur) fera
// évoluer cette structure pour gérer plusieurs joueurs avec FOV par joueur
// et focus des monstres côté serveur.
type Server struct {
	state *game.State
	ln    net.Listener
}

// NewServer crée un serveur avec une nouvelle partie (seed, dimensions).
func NewServer(seed int64, width, height int) *Server {
	return &Server{state: game.NewStateSized(seed, width, height)}
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
// une goroutine dédiée. En bloc 5, chaque client joue sa propre partie solo
// sur le même state (mono-joueur distant) ; le multi viendra en bloc 6.
func (s *Server) Serve() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// handleConn sert un client : envoie le snapshot initial, puis boucle sur les
// inputs (applique → snapshot) jusqu'à la fin de partie ou la déconnexion.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	s.sendSnapshot(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return // déconnexion
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
		s.applyInput(in)
		s.sendSnapshot(conn)
		if s.state.Phase != game.PhasePlaying {
			s.sendEnd(conn, "partie terminée")
			return
		}
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

// sendSnapshot envoie l'état visible au client (JSON + newline).
func (s *Server) sendSnapshot(conn net.Conn) {
	env, err := netproto.Encode(netproto.MsgSnapshot, netproto.SnapshotPayload{Snap: s.state.Snapshot()})
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
