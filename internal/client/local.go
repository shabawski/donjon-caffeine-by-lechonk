// Package client définit l'interface GameClient utilisée par le TUI pour
// communiquer avec le serveur de jeu, ainsi que les implémentations :
// LocalClient (solo, serveur embarqué via channels) et TCPClient (multi, TCP).
package client

import (
	"game-test/internal/game"
	"game-test/internal/net"
)

// GameClient est l'interface que le TUI utilise pour communiquer avec le
// serveur. Le TUI envoie des inputs, reçoit des snapshots, et rend depuis
// le dernier snapshot reçu. L'implémentation choisie (local ou TCP) est
// transparente pour le TUI.
type GameClient interface {
	// Connect établit la connexion (no-op pour le local).
	Connect() error
	// SendInput envoie une action du joueur au serveur (non bloquant).
	SendInput(in net.InputPayload) error
	// NextSnapshot bloque jusqu'au prochain snapshot du serveur.
	// Retourne (nil, nil) si la partie est terminée.
	NextSnapshot() (*game.Snapshot, error)
	// DrainSnapshots vide les snapshots en attente sans bloquer.
	// En multi, écarte les broadcasts des autres joueurs avant un input.
	DrainSnapshots()
	// Close libère les ressources.
	Close() error
}

// LocalClient joue en solo : le serveur tourne dans une goroutine du même
// process et communique par channels. Même interface que le TCPClient.
type LocalClient struct {
	state  *game.State
	inCh   chan net.InputPayload // client → serveur
	outCh  chan *game.Snapshot   // serveur → client
	done   chan struct{}
	closed bool
}

// NewLocalClient crée un client solo avec une partie neuve (seed, dimensions).
// Le serveur embarqué envoie le snapshot initial immédiatement.
func NewLocalClient(seed int64, width, height int) *LocalClient {
	return NewLocalClientMode(seed, width, height, game.ModeCoop)
}

// NewLocalClientMode crée un client solo avec le mode de jeu donné
// (ModeCoop = donjon, ModeMMORPG = monde ouvert).
func NewLocalClientMode(seed int64, width, height int, mode game.GameMode) *LocalClient {
	c := &LocalClient{
		inCh:  make(chan net.InputPayload, 16),
		outCh: make(chan *game.Snapshot, 16),
		done:  make(chan struct{}),
	}
	if mode == game.ModeMMORPG {
		c.state = game.NewStateMMORPG(seed, width, height)
	} else {
		c.state = game.NewStateSized(seed, width, height)
	}
	go c.serverLoop()
	return c
}

// serverLoop applique les inputs sur le state et publie les snapshots.
func (c *LocalClient) serverLoop() {
	c.outCh <- c.state.Snapshot()
	for {
		select {
		case in := <-c.inCh:
			c.applyInput(in)
			c.outCh <- c.state.Snapshot()
			if c.state.Phase != game.PhasePlaying {
				return // fin de partie : le dernier snapshot a été envoyé
			}
		case <-c.done:
			return
		}
	}
}

// applyInput applique une action du joueur sur le state (côté serveur).
func (c *LocalClient) applyInput(in net.InputPayload) {
	s := c.state
	switch in.Kind {
	case net.InputMove:
		s.MovePlayer(in.DX, in.DY)
	case net.InputWait:
		s.PlayerWait()
	case net.InputFireball:
		s.CastFireball(in.LastDX, in.LastDY)
	case net.InputDescend:
		s.Descend()
	case net.InputAscend:
		s.Ascend()
	case net.InputEquip:
		s.EquipItem(in.Index)
	case net.InputUse:
		s.UseItem(in.Index)
	case net.InputDrop:
		s.DropItem(in.Index)
	case net.InputBuy:
		s.BuyItem(in.Index)
	case net.InputSell:
		s.SellItem(in.Index)
	case net.InputEnterBuilding:
		s.EnterBuilding()
	case net.InputExitBuilding:
		s.ExitBuilding()
	}
}

// Connect est un no-op en local (le serveur démarre avec NewLocalClient).
func (c *LocalClient) Connect() error { return nil }

// SendInput envoie une action au serveur embarqué (bufferisé, non bloquant).
func (c *LocalClient) SendInput(in net.InputPayload) error {
	c.inCh <- in
	return nil
}

// NextSnapshot bloque jusqu'au prochain snapshot. Retourne (nil, nil) si la
// partie est terminée (le dernier snapshot a déjà été reçu avant).
func (c *LocalClient) NextSnapshot() (*game.Snapshot, error) {
	select {
	case snap := <-c.outCh:
		return snap, nil
	case <-c.done:
		return nil, nil
	}
}

// Close arrête le serveur embarqué.
func (c *LocalClient) Close() error {
	if !c.closed {
		c.closed = true
		close(c.done)
	}
	return nil
}

// DrainSnapshots vide les snapshots en attente sans bloquer (no-op utile en
// solo — un seul joueur, pas de broadcast — mais requis par l'interface).
func (c *LocalClient) DrainSnapshots() {
	for {
		select {
		case <-c.outCh:
		default:
			return
		}
	}
}

// State expose l'état interne du serveur embarqué (pour les tests uniquement).
func (c *LocalClient) State() *game.State { return c.state }
