package client

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"

	"game-test/internal/game"
	netproto "game-test/internal/net"
)

// tcpBufferSize : taille du buffer de lecture (1 Mo). Un Snapshot complet
// (carte 80x40 + FOV + entités + journal) fait ~50 Ko en JSON ; 1 Mo laisse
// de la marge pour les grandes cartes et les inventaires chargés.
const tcpBufferSize = 1024 * 1024

// TCPClient joue en multijoueur : connexion TCP à un serveur distant
// (cmd/coffee-dungeon-server). Même interface GameClient que LocalClient.
// Le protocole est du JSON délimité par des sauts de ligne (un Envelope par ligne).
//
// Thread-safety : une seule goroutine (readLoop) lit sur le bufio.Reader
// (bufio n'est PAS thread-safe — plusieurs lecteurs simultanés provoquent
// un panic "slice bounds out of range"). Les snapshots sont poussés dans un
// channel bufferisé ; NextSnapshot lit depuis ce channel (thread-safe).
// En multijoueur, le serveur broadcast un snapshot à chaque client après
// chaque input : le channel accumule ces broadcasts, consommés au rythme
// du TUI (qui relance waitSnapshot après chaque snapshot reçu).
type TCPClient struct {
	addr      string
	conn      net.Conn
	snapshots chan *game.Snapshot // snapshots reçus (bufferisé)
	errs      chan error          // erreur de lecture (capacité 1)
	done      chan struct{}       // fermeture
	closeOnce sync.Once
}

// NewTCPClient crée un client TCP (la connexion est établie par Connect).
func NewTCPClient(addr string) (*TCPClient, error) {
	return &TCPClient{
		addr: addr,
	}, nil
}

// Connect établit la connexion TCP et lance la goroutine de lecture.
func (c *TCPClient) Connect() error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.snapshots = make(chan *game.Snapshot, 64)
	c.errs = make(chan error, 1)
	c.done = make(chan struct{})
	go c.readLoop(bufio.NewReaderSize(conn, tcpBufferSize))
	return nil
}

// readLoop lit les lignes JSON sur la connexion (seule goroutine lectrice)
// et pousse les snapshots dans le channel. Se termine à la déconnexion ou
// à la réception d'un MsgEnd (ferme le channel de snapshots).
func (c *TCPClient) readLoop(reader *bufio.Reader) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// Déconnexion ou erreur de lecture : signaler (non bloquant).
			select {
			case c.errs <- err:
			default:
			}
			close(c.snapshots)
			return
		}
		var env netproto.Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue // ignore les lignes mal formées
		}
		switch env.Type {
		case netproto.MsgSnapshot:
			var payload netproto.SnapshotPayload
			if err := netproto.DecodePayload(env, &payload); err != nil || payload.Snap == nil {
				continue
			}
			select {
			case c.snapshots <- payload.Snap:
			case <-c.done:
				return
			}
		case netproto.MsgEnd:
			close(c.snapshots) // fin de partie
			return
		default:
			// ignore les autres types (Welcome, etc.)
		}
	}
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

// NextSnapshot bloque jusqu'au prochain snapshot du serveur (thread-safe :
// lit depuis le channel alimenté par readLoop).
// Retourne (nil, nil) si le serveur a envoyé MsgEnd (fin de partie) ou si
// le client est fermé.
func (c *TCPClient) NextSnapshot() (*game.Snapshot, error) {
	select {
	case snap, ok := <-c.snapshots:
		if !ok {
			return nil, nil // channel fermé : fin de partie
		}
		return snap, nil
	case err := <-c.errs:
		return nil, err
	case <-c.done:
		return nil, nil
	}
}

// DrainSnapshots vide les snapshots en attente sans bloquer.
// Utilisé avant d'envoyer un input en multi : écarte les broadcasts des
// autres joueurs pour que le prochain NextSnapshot corresponde à NOTRE input.
func (c *TCPClient) DrainSnapshots() {
	for {
		select {
		case <-c.snapshots:
		default:
			return
		}
	}
}

// Close ferme la connexion TCP (idempotent).
func (c *TCPClient) Close() error {
	c.closeOnce.Do(func() {
		if c.done != nil {
			close(c.done)
		}
		if c.conn != nil {
			c.conn.Close()
		}
	})
	return nil
}
