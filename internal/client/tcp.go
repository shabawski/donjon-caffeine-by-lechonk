package client

import (
	"bufio"
	"encoding/json"
	"net"

	"game-test/internal/game"
	netproto "game-test/internal/net"
)

// TCPClient joue en multijoueur : connexion TCP à un serveur distant
// (cmd/coffee-dungeon-server). Même interface GameClient que LocalClient.
// Le protocole est du JSON délimité par des sauts de ligne (un Envelope par ligne).
type TCPClient struct {
	addr   string
	conn   net.Conn
	reader *bufio.Reader
}

// NewTCPClient crée un client TCP (la connexion est établie par Connect).
func NewTCPClient(addr string) (*TCPClient, error) {
	return &TCPClient{addr: addr}, nil
}

// Connect établit la connexion TCP avec le serveur.
func (c *TCPClient) Connect() error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

// SendInput envoie une action du joueur au serveur (JSON + newline).
func (c *TCPClient) SendInput(in netproto.InputPayload) error {
	env, err := netproto.Encode(netproto.MsgInput, in)
	if err != nil {
		return err
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(append(data, '\n'))
	return err
}

// NextSnapshot bloque jusqu'au prochain snapshot du serveur.
// Retourne (nil, nil) si le serveur envoie MsgEnd (fin de partie).
func (c *TCPClient) NextSnapshot() (*game.Snapshot, error) {
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		var env netproto.Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue // ignore les lignes mal formées
		}
		switch env.Type {
		case netproto.MsgSnapshot:
			var payload netproto.SnapshotPayload
			if err := netproto.DecodePayload(env, &payload); err != nil {
				return nil, err
			}
			return payload.Snap, nil
		case netproto.MsgEnd:
			return nil, nil // fin de partie
		default:
			// ignore les autres types (Welcome, etc.)
		}
	}
}

// Close ferme la connexion TCP.
func (c *TCPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
