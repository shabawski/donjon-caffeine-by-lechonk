package client

import (
	"testing"
	"time"

	"game-test/internal/game"
	netproto "game-test/internal/net"
	"game-test/internal/server"
)

// TestServerMultiClient lance un serveur TCP et connecte deux clients.
// Vérifie que chacun reçoit un snapshot personnalisé (PlayerID distinct),
// et que le broadcast après un input atteint les deux clients.
func TestServerMultiClient(t *testing.T) {
	srv := server.NewServer(42, game.DefaultWidth, game.DefaultHeight)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	// Connecte deux clients TCP.
	c1, err := NewTCPClient(srv.Addr())
	if err != nil {
		t.Fatalf("c1 NewTCPClient: %v", err)
	}
	if err := c1.Connect(); err != nil {
		t.Fatalf("c1 Connect: %v", err)
	}
	defer c1.Close()
	c2, err := NewTCPClient(srv.Addr())
	if err != nil {
		t.Fatalf("c2 NewTCPClient: %v", err)
	}
	if err := c2.Connect(); err != nil {
		t.Fatalf("c2 Connect: %v", err)
	}
	defer c2.Close()

	// Chaque client reçoit un snapshot initial avec un PlayerID distinct.
	snap1, err := c1.NextSnapshot()
	if err != nil || snap1 == nil {
		t.Fatalf("c1 snapshot initial: %v (%v)", snap1, err)
	}
	snap2, err := c2.NextSnapshot()
	if err != nil || snap2 == nil {
		t.Fatalf("c2 snapshot initial: %v (%v)", snap2, err)
	}
	if snap1.PlayerID == snap2.PlayerID {
		t.Fatalf("les deux clients doivent avoir des PlayerID distincts: %d", snap1.PlayerID)
	}

	// c1 envoie un input (wait). Le serveur broadcast à tous les clients.
	if err := c1.SendInput(netproto.InputPayload{Kind: netproto.InputWait}); err != nil {
		t.Fatalf("c1 SendInput: %v", err)
	}
	// c1 reçoit un snapshot (broadcast) : son tour a augmenté.
	snap1b, err := c1.NextSnapshot()
	if err != nil || snap1b == nil {
		t.Fatalf("c1 snapshot après input: %v (%v)", snap1b, err)
	}
	if snap1b.Player.Turns != snap1.Player.Turns+1 {
		t.Fatalf("c1: tour attendu %d, got %d", snap1.Player.Turns+1, snap1b.Player.Turns)
	}
	// c2 reçoit aussi un snapshot (broadcast) et voit c1 dans ses Others.
	snap2b, err := c2.NextSnapshot()
	if err != nil || snap2b == nil {
		t.Fatalf("c2 snapshot après input c1: %v (%v)", snap2b, err)
	}
	if len(snap2b.Others) == 0 {
		t.Fatal("c2 doit voir c1 dans Others après connexion de c1")
	}
}

// TestServerMultiClientConcurrent envoie des inputs depuis deux clients
// et vérifie que le serveur ne panique pas et répond aux deux (mutex OK).
func TestServerMultiClientConcurrent(t *testing.T) {
	srv := server.NewServer(7, 40, 20)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	connect := func() *TCPClient {
		c, err := NewTCPClient(srv.Addr())
		if err != nil {
			t.Fatalf("NewTCPClient: %v", err)
		}
		if err := c.Connect(); err != nil {
			t.Fatalf("Connect: %v", err)
		}
		return c
	}
	c1, c2 := connect(), connect()
	defer c1.Close()
	defer c2.Close()

	// Snapshots initiaux.
	if _, err := c1.NextSnapshot(); err != nil {
		t.Fatalf("c1 initial: %v", err)
	}
	if _, err := c2.NextSnapshot(); err != nil {
		t.Fatalf("c2 initial: %v", err)
	}

	// Envoie 5 inputs depuis chaque client (intercalés).
	for i := 0; i < 5; i++ {
		_ = c1.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
		_ = c2.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
	}
	// Chaque client doit recevoir au moins 5 snapshots (broadcast à chaque input).
	for i := 0; i < 5; i++ {
		if _, err := c1.NextSnapshot(); err != nil {
			t.Fatalf("c1 snapshot %d: %v", i, err)
		}
	}
	for i := 0; i < 5; i++ {
		if _, err := c2.NextSnapshot(); err != nil {
			t.Fatalf("c2 snapshot %d: %v", i, err)
		}
	}
}

// TestServerClientDisconnect vérifie que la déconnexion d'un client ne
// bloque pas le serveur (l'autre client continue de recevoir des snapshots).
func TestServerClientDisconnect(t *testing.T) {
	srv := server.NewServer(99, 40, 20)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	c1, _ := NewTCPClient(srv.Addr())
	if err := c1.Connect(); err != nil {
		t.Fatalf("c1 Connect: %v", err)
	}
	c2, _ := NewTCPClient(srv.Addr())
	if err := c2.Connect(); err != nil {
		t.Fatalf("c2 Connect: %v", err)
	}
	defer c2.Close()

	// Snapshots initiaux.
	_, _ = c1.NextSnapshot()
	snap2, _ := c2.NextSnapshot()
	if snap2 == nil {
		t.Fatal("c2 snapshot initial nil")
	}

	// c1 se déconnecte brutalement.
	c1.Close()

	// c2 envoie un input : le serveur doit continuer à répondre.
	_ = c2.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
	done := make(chan struct{})
	go func() {
		_, err := c2.NextSnapshot()
		if err != nil {
			t.Errorf("c2 après déconnexion c1: %v", err)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: c2 n'a pas reçu de snapshot après déconnexion de c1")
	}
}
