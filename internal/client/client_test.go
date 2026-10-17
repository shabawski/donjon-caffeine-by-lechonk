package client

import (
	"testing"

	"game-test/internal/game"
	netproto "game-test/internal/net"
	"game-test/internal/server"
)

// TestTCPClientServerRoundTrip lance un serveur TCP sur un port aléatoire,
// connecte un TCPClient, et vérifie l'échange snapshot/input/snapshot.
func TestTCPClientServerRoundTrip(t *testing.T) {
	// Lance un serveur sur un port aléatoire (127.0.0.1:0 = port auto).
	srv := server.NewServer(42, game.DefaultWidth, game.DefaultHeight)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve() // accepte les connexions en arrière-plan

	// Connecte un client TCP.
	c, err := NewTCPClient(srv.Addr())
	if err != nil {
		t.Fatalf("NewTCPClient: %v", err)
	}
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	// Reçoit le snapshot initial.
	snap, err := c.NextSnapshot()
	if err != nil {
		t.Fatalf("NextSnapshot initial: %v", err)
	}
	if snap == nil {
		t.Fatal("snapshot initial nil")
	}
	if snap.Seed != 42 {
		t.Fatalf("seed attendue 42, got %d", snap.Seed)
	}
	if snap.Depth != 1 {
		t.Fatalf("étage attendu 1, got %d", snap.Depth)
	}

	// Envoie un input (attendre un tour).
	if err := c.SendInput(netproto.InputPayload{Kind: netproto.InputWait}); err != nil {
		t.Fatalf("SendInput: %v", err)
	}

	// Reçoit le snapshot suivant : le tour doit avoir augmenté.
	snap2, err := c.NextSnapshot()
	if err != nil {
		t.Fatalf("NextSnapshot après input: %v", err)
	}
	if snap2 == nil {
		t.Fatal("snapshot après input nil")
	}
	if snap2.Player.Turns != snap.Player.Turns+1 {
		t.Fatalf("tour attendu %d, got %d", snap.Player.Turns+1, snap2.Player.Turns)
	}
}

// TestLocalClientRoundTrip vérifie le client solo (channels) : input → snapshot.
func TestLocalClientRoundTrip(t *testing.T) {
	c := NewLocalClient(42, game.DefaultWidth, game.DefaultHeight)
	defer c.Close()

	// Snapshot initial.
	snap, err := c.NextSnapshot()
	if err != nil {
		t.Fatalf("NextSnapshot initial: %v", err)
	}
	if snap == nil || snap.Seed != 42 {
		t.Fatalf("snapshot initial incorrect: %v", snap)
	}

	// Envoie un input (attendre un tour) et reçoit le snapshot.
	if err := c.SendInput(netproto.InputPayload{Kind: netproto.InputWait}); err != nil {
		t.Fatalf("SendInput: %v", err)
	}
	snap2, err := c.NextSnapshot()
	if err != nil {
		t.Fatalf("NextSnapshot après input: %v", err)
	}
	if snap2.Player.Turns != snap.Player.Turns+1 {
		t.Fatalf("tour attendu %d, got %d", snap.Player.Turns+1, snap2.Player.Turns)
	}
}

// TestLocalClientDeterministic vérifie que deux clients avec la même seed
// produisent les mêmes snapshots après les mêmes inputs.
func TestLocalClientDeterministic(t *testing.T) {
	a := NewLocalClient(99, game.DefaultWidth, game.DefaultHeight)
	b := NewLocalClient(99, game.DefaultWidth, game.DefaultHeight)
	defer a.Close()
	defer b.Close()

	sa, _ := a.NextSnapshot()
	sb, _ := b.NextSnapshot()
	if sa.Player.X != sb.Player.X || sa.Player.Y != sb.Player.Y {
		t.Fatal("même seed, positions de départ différentes")
	}

	// Même input des deux côtés.
	_ = a.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
	_ = b.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
	sa2, _ := a.NextSnapshot()
	sb2, _ := b.NextSnapshot()
	if sa2.Player.Turns != sb2.Player.Turns {
		t.Fatalf("même seed, tours différents: %d vs %d", sa2.Player.Turns, sb2.Player.Turns)
	}
	if sa2.Score() != sb2.Score() {
		t.Fatalf("même seed, scores différents: %d vs %d", sa2.Score(), sb2.Score())
	}
}
