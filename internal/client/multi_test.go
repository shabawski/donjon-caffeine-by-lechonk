package client

import (
	"encoding/json"
	"io"
	"sync"
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

// TestTCPClientConcurrentNextSnapshot — RÉGRESSION (17/10) : plusieurs
// NextSnapshot simultanés ne doivent pas paniquer. Avant le fix, chaque
// appel lisait sur le même bufio.Reader (non thread-safe) → panic
// "slice bounds out of range". Le fix : readLoop (goroutine unique) + channel.
func TestTCPClientConcurrentNextSnapshot(t *testing.T) {
	srv := server.NewServer(42, game.DefaultWidth, game.DefaultHeight)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	c, _ := NewTCPClient(srv.Addr())
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	// Consomme le snapshot initial.
	if _, err := c.NextSnapshot(); err != nil {
		t.Fatalf("snapshot initial: %v", err)
	}

	// Envoie 10 inputs rapidement (le serveur broadcast 10 snapshots).
	const n = 10
	for i := 0; i < n; i++ {
		_ = c.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
	}
	// Appelle NextSnapshot en parallèle depuis 10 goroutines : aucun panic.
	// Note : snap == nil est valide (fin de partie si le joueur meurt pendant
	// les 10 tours "wait" — le serveur envoie alors MsgEnd et ferme le channel).
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.NextSnapshot()
			// EOF / nil sont valides : fin de partie (le joueur peut mourir
			// pendant les 10 tours "wait" → le serveur envoie MsgEnd puis
			// ferme la connexion). Le but du test est l'absence de panic.
			if err != nil && err != io.EOF {
				t.Errorf("NextSnapshot: %v", err)
			}
		}()
	}
	wg.Wait()
}

// TestTCPClientLargeSnapshot — RÉGRESSION (17/10) : un snapshot JSON
// dépassant le buffer bufio par défaut (4096 octets) doit être lu sans
// erreur. Le buffer de lecture est porté à 1 Mo.
func TestTCPClientLargeSnapshot(t *testing.T) {
	srv := server.NewServer(42, 200, 100) // grande carte = gros snapshot JSON
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	c, _ := NewTCPClient(srv.Addr())
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	snap, err := c.NextSnapshot()
	if err != nil {
		t.Fatalf("snapshot initial: %v", err)
	}
	if snap == nil {
		t.Fatal("snapshot nil")
	}
	if snap.Width != 200 || snap.Height != 100 {
		t.Fatalf("dimensions attendues 200x100, got %dx%d", snap.Width, snap.Height)
	}
	// Vérifie que le JSON dépasse bien 4096 octets (sinon le test ne prouve rien).
	data, _ := json.Marshal(snap)
	if len(data) <= 4096 {
		t.Fatalf("snapshot trop petit pour le test de dépassement de buffer: %d octets", len(data))
	}
	t.Logf("snapshot JSON: %d octets (dépasse le buffer bufio par défaut de 4096)", len(data))
}

// TestServerSlowClientDoesNotBlockOthers — RÉGRESSION (17/10 15:20) : un
// client lent (ne consomme pas ses snapshots) ne doit pas bloquer le serveur.
// Avant le fix, broadcastSnapshotsLocked écrivait sur les connexions TCP en
// tenant le mutex global → un client lent bloquait TOUT le serveur (l'autre
// joueur ne recevait plus rien, "bloqué, les inputs ne répondent plus").
// Après le fix : outbox par client + writeLoop, broadcast hors mutex, client
// trop lent déconnecté (outbox plein).
func TestServerSlowClientDoesNotBlockOthers(t *testing.T) {
	srv := server.NewServer(42, 40, 20)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	// Client 1 : actif (consomme ses snapshots).
	c1, _ := NewTCPClient(srv.Addr())
	if err := c1.Connect(); err != nil {
		t.Fatalf("c1 connect: %v", err)
	}
	defer c1.Close()
	if _, err := c1.NextSnapshot(); err != nil {
		t.Fatalf("c1 initial: %v", err)
	}

	// Client 2 : lent (lit le snapshot initial, puis ne consomme PLUS RIEN).
	c2, _ := NewTCPClient(srv.Addr())
	if err := c2.Connect(); err != nil {
		t.Fatalf("c2 connect: %v", err)
	}
	defer c2.Close()
	if _, err := c2.NextSnapshot(); err != nil {
		t.Fatalf("c2 initial: %v", err)
	}

	// Envoie 300 inputs depuis c1. L'outbox de c2 fait 256 → c2 sera déconnecté
	// par le serveur (trop lent). c1 doit recevoir les snapshots sans blocage.
	const n = 300
	for i := 0; i < n; i++ {
		_ = c1.SendInput(netproto.InputPayload{Kind: netproto.InputWait})
	}

	// Lit les snapshots de c1 jusqu'à épuisement ou fin de partie (le joueur
	// peut mourir pendant les 300 tours "wait" — les monstres l'attaquent).
	snapCh := make(chan *game.Snapshot, n)
	go func() {
		for i := 0; i < n; i++ {
			snap, err := c1.NextSnapshot()
			if err != nil {
				t.Errorf("c1 snapshot %d: %v", i, err)
				return
			}
			snapCh <- snap
			if snap == nil {
				return // fin de partie (MsgEnd)
			}
		}
	}()
	received := 0
	deadline := time.After(10 * time.Second)
	for received < n {
		select {
		case snap := <-snapCh:
			received++
			if snap == nil {
				t.Logf("fin de partie après %d snapshots (joueur mort)", received)
				return // OK : le serveur a répondu jusqu'à la fin
			}
		case <-deadline:
			t.Fatalf("timeout: c1 n'a reçu que %d/%d snapshots (serveur bloqué par c2 lent ?)", received, n)
		}
	}
	t.Logf("c1 a reçu %d snapshots sans blocage (c2 lent géré)", received)
}
