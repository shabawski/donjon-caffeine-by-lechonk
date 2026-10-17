package tui

import (
	"strings"
	"testing"
	"time"

	"game-test/internal/client"
	"game-test/internal/game"
	netproto "game-test/internal/net"
	"game-test/internal/score"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel crée un Model adossé à un LocalClient (serveur solo embarqué).
// directPlay=false : démarre sur le menu (comme en solo sans --connect).
func newTestModel(seed int64) (*Model, *client.LocalClient) {
	c := client.NewLocalClient(seed, game.DefaultWidth, game.DefaultHeight)
	snap, _ := c.NextSnapshot()
	return New(c, snap, false), c
}

// press envoie une touche au modèle et consomme le snapshot de réponse si un
// input a été envoyé. Les commandes bloquantes (waitSnapshot) sont exécutées
// dans une goroutine avec un timeout court : elles sont ignorées si elles ne
// produisent pas de message rapidement (pas de boucle BubbleTea dans les tests).
func press(m *Model, key tea.KeyMsg) *Model {
	updated, cmd := m.Update(key)
	m = updated.(*Model)
	if cmd != nil {
		type result struct{ msg tea.Msg }
		ch := make(chan result, 1)
		go func() { ch <- result{cmd()} }()
		select {
		case r := <-ch:
			if r.msg != nil {
				updated, _ = m.Update(r.msg)
				m = updated.(*Model)
			}
		case <-time.After(100 * time.Millisecond):
			// Commande bloquante (waitSnapshot) : ignorée.
		}
	}
	// Si un input a été envoyé, consomme le snapshot de réponse.
	if m.pendingInput {
		snap, err := m.client.NextSnapshot()
		if err == nil && snap != nil {
			updated, _ = m.Update(snapshotMsg{snap})
			m = updated.(*Model)
		}
	}
	return m
}

// mockClient retourne toujours le même snapshot (pour les tests d'écran).
type mockClient struct {
	snap   *game.Snapshot
	closed bool // true si Close() a été appelé (test joueurs fantômes)
}

func (c *mockClient) Connect() error                        { return nil }
func (c *mockClient) SendInput(netproto.InputPayload) error { return nil }
func (c *mockClient) NextSnapshot() (*game.Snapshot, error) { return c.snap, nil }
func (c *mockClient) DrainSnapshots()                       {}
func (c *mockClient) Close() error                          { c.closed = true; return nil }

func TestViewMenuContainsTitle(t *testing.T) {
	m, _ := newTestModel(42)
	if v := m.View(); !strings.Contains(v, "DONJON") {
		t.Fatalf("titre du menu absent: %q", v)
	}
}

func TestMenuEnterStartsGame(t *testing.T) {
	m, _ := newTestModel(42)
	// enter sur "Nouvelle partie" retourne une cmd (startNewGame) → newGameMsg
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatal("enter doit lancer une commande de nouvelle partie")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("la commande doit produire un newGameMsg")
	}
	updated, _ = m.Update(msg)
	m = updated.(*Model)
	if m.screen != screenPlaying {
		t.Fatal("enter doit lancer la partie depuis le menu")
	}
	if m.snapshot == nil {
		t.Fatal("le snapshot initial doit être reçu")
	}
}

func TestMenuQuit(t *testing.T) {
	m, _ := newTestModel(42)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q doit produire tea.Quit depuis le menu")
	}
}

// TestMenuMMORPGStartsGame vérifie que l'option 1 (MMORPG) lance une partie
// en monde ouvert : snapshot.Mode == MMORPG et titre "Monde ouvert".
func TestMenuMMORPGStartsGame(t *testing.T) {
	m, _ := newTestModel(42)
	// Descend jusqu'à l'option 1 (MMORPG).
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.menuCursor != 1 {
		t.Fatalf("curseur attendu 1, got %d", m.menuCursor)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatal("enter doit lancer une commande de nouvelle partie")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("la commande doit produire un newGameMsg")
	}
	updated, _ = m.Update(msg)
	m = updated.(*Model)
	if m.screen != screenPlaying {
		t.Fatal("enter doit lancer la partie depuis le menu")
	}
	if m.snapshot.Mode != game.ModeMMORPG {
		t.Fatalf("mode attendu MMORPG, got %v", m.snapshot.Mode)
	}
	if m.mode != game.ModeMMORPG {
		t.Fatalf("le modèle doit conserver le mode MMORPG, got %v", m.mode)
	}
	if v := m.View(); !strings.Contains(v, "Monde ouvert") {
		t.Fatalf("le titre doit indiquer le monde ouvert: %q", v)
	}
}

func TestMenuNavigationWraps(t *testing.T) {
	m, _ := newTestModel(42)
	m.Update(tea.KeyMsg{Type: tea.KeyUp}) // remonte au-dessus de 0 : reste à 0
	if m.menuCursor != 0 {
		t.Fatalf("curseur attendu 0, got %d", m.menuCursor)
	}
	for i := 0; i < 7; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.menuCursor != 4 {
		t.Fatalf("curseur plafonné à 4 (5 options), got %d", m.menuCursor)
	}
}

// TestMenuMultiOuvreConnect vérifie que l'option "Multijoueur (TCP)" ouvre
// l'écran de saisie de l'adresse serveur.
func TestMenuMultiOuvreConnect(t *testing.T) {
	m, _ := newTestModel(42)
	// Descend jusqu'à l'option 2 (Multijoueur).
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.menuCursor != 2 {
		t.Fatalf("curseur attendu 2, got %d", m.menuCursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenConnect {
		t.Fatalf("enter sur Multijoueur doit ouvrir l'écran connexion, got %v", m.screen)
	}
	if v := m.View(); !strings.Contains(v, "host:port") {
		t.Fatal("l'écran connexion doit afficher le champ host:port")
	}
}

// TestConnectSaisieEtRetour vérifie la saisie de l'adresse et le retour menu.
func TestConnectSaisieEtRetour(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenConnect
	// Saisit "127.0.0.1:27015" caractère par caractère.
	for _, r := range "127.0.0.1:27015" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.connectInput != "127.0.0.1:27015" {
		t.Fatalf("adresse saisie incorrecte: %q", m.connectInput)
	}
	// Backspace supprime le dernier caractère.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.connectInput != "127.0.0.1:2701" {
		t.Fatalf("backspace incorrect: %q", m.connectInput)
	}
	// Esc retourne au menu.
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenMenu {
		t.Fatal("esc doit ramener au menu depuis l'écran connexion")
	}
}

func TestPlayingMoveKey(t *testing.T) {
	m, c := newTestModel(42)
	m.screen = screenPlaying
	// Place le joueur sur une case avec la droite libre (via le state du serveur).
	s := c.State()
	found := false
outer:
	for y := 0; y < s.CurrentMap().World.Height; y++ {
		for x := 0; x+1 < s.CurrentMap().World.Width; x++ {
			if s.CurrentMap().World.IsWalkable(x, y) && s.CurrentMap().World.IsWalkable(x+1, y) && s.OccupiedByEnemy(x+1, y) == game.InvalidEntity {
				s.Player.X, s.Player.Y = x, y
				found = true
				break outer
			}
		}
	}
	if !found {
		t.Fatal("pas de case libre trouvée")
	}
	x0 := s.Player.X
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.snapshot.Player.X != x0+1 {
		t.Fatalf("la touche l doit déplacer à droite: %d -> %d", x0, m.snapshot.Player.X)
	}
}

func TestViewGameContainsPlayerGlyphAndHUD(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenPlaying
	v := m.View()
	if !strings.Contains(v, "@") {
		t.Fatal("le glyph du joueur doit être rendu")
	}
	if !strings.Contains(v, "PV") {
		t.Fatal("le HUD doit afficher les PV")
	}
	if !strings.Contains(v, "étage 1/10") {
		t.Fatal("le titre doit indiquer l'étage courant")
	}
}

func TestHelpOverlayRoundTrip(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenPlaying
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.screen != screenHelp {
		t.Fatal("? doit ouvrir l'aide")
	}
	if v := m.View(); !strings.Contains(v, "boule de feu") {
		t.Fatal("l'aide doit lister les commandes")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenPlaying {
		t.Fatal("esc doit fermer l'aide")
	}
}

func TestInventoryOverlayRoundTrip(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenPlaying
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.screen != screenInventory {
		t.Fatal("i doit ouvrir l'inventaire")
	}
	if v := m.View(); !strings.Contains(v, "Inventaire") {
		t.Fatal("l'inventaire doit s'afficher")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.screen != screenPlaying {
		t.Fatal("i doit fermer l'inventaire")
	}
}

func TestQuitFromGameReturnsToMenu(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenPlaying
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenMenu {
		t.Fatal("esc en jeu doit ramener au menu")
	}
	// Q majuscule fait de même.
	m.screen = screenPlaying
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	if m.screen != screenMenu {
		t.Fatal("Q en jeu doit ramener au menu")
	}
}

func TestZQSDQMovesLeft(t *testing.T) {
	m, c := newTestModel(42)
	m.screen = screenPlaying
	s := c.State()
	found := false
outer:
	for y := 0; y < s.CurrentMap().World.Height; y++ {
		for x := 1; x < s.CurrentMap().World.Width; x++ {
			if s.CurrentMap().World.IsWalkable(x, y) && s.CurrentMap().World.IsWalkable(x-1, y) && s.OccupiedByEnemy(x-1, y) == game.InvalidEntity {
				s.Player.X, s.Player.Y = x, y
				found = true
				break outer
			}
		}
	}
	if !found {
		t.Fatal("pas de case libre trouvée")
	}
	x0 := s.Player.X
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.snapshot.Player.X != x0-1 {
		t.Fatalf("la touche q (ZQSD) doit déplacer à gauche: %d -> %d", x0, m.snapshot.Player.X)
	}
}

func TestDeathTriggersGameOverAndSavesScore(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := game.NewState(42)
	s.Player.HP = 0
	s.PlayerDeath()
	snap := s.Snapshot()
	m := New(&mockClient{snap: snap}, snap, false)
	m.screen = screenPlaying
	m.afterSnapshot() // simule la réception du snapshot
	if m.screen != screenGameOver {
		t.Fatal("écran game over attendu après la mort")
	}
	if !m.scoreSaved {
		t.Fatal("le score doit être marqué sauvegardé")
	}
	entries, err := score.Load()
	if err != nil || len(entries) != 1 {
		t.Fatalf("le score doit être persisté: %v (%d entrées)", err, len(entries))
	}
	if v := m.View(); !strings.Contains(v, "GAME OVER") {
		t.Fatal("l'écran de fin doit annoncer la défaite")
	}
}

func TestVictoryScreen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := game.NewState(42)
	// Transforme le premier monstre en boss avec 1 PV (registre ECS/SOA).
	if s.CurrentMap().Entities.Len() == 0 {
		t.Fatal("aucun monstre disponible pour le test de victoire")
	}
	s.CurrentMap().Entities.Kind[0] = game.KindBoss
	s.CurrentMap().Entities.Name[0] = "Seigneur des Tasses"
	s.CurrentMap().Entities.HP[0] = 1
	s.Player.Attack = 1000
	s.Player.X, s.Player.Y = s.CurrentMap().Entities.X[0], s.CurrentMap().Entities.Y[0]
	s.MovePlayer(0, 0) // attaque le boss → victoire
	snap := s.Snapshot()
	m := New(&mockClient{snap: snap}, snap, false)
	m.screen = screenPlaying
	m.afterSnapshot()
	if m.screen != screenGameOver {
		t.Fatal("écran de fin attendu après la victoire")
	}
	if v := m.View(); !strings.Contains(v, "VICTOIRE") {
		t.Fatalf("l'écran de fin doit annoncer la victoire: %q", v)
	}
}

func TestGameOverRejouerStartsNewGame(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, c := newTestModel(42)
	m.screen = screenGameOver
	m.scoreSaved = true
	oldSeed := m.snapshot.Seed
	// r retourne une cmd (startNewGame) → newGameMsg
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(*Model)
	if cmd == nil {
		t.Fatal("r doit lancer une commande de nouvelle partie")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(*Model)
	if m.screen != screenPlaying {
		t.Fatal("r doit relancer une partie")
	}
	if m.scoreSaved {
		t.Fatal("le flag scoreSaved doit être remis à zéro")
	}
	if m.snapshot.Seed == oldSeed {
		t.Fatal("une nouvelle partie doit avoir une nouvelle graine")
	}
	_ = c
}

func TestDirFromKey(t *testing.T) {
	cases := map[string][2]int{
		"up": {0, -1}, "k": {0, -1}, "z": {0, -1},
		"down": {0, 1}, "j": {0, 1}, "s": {0, 1},
		"left": {-1, 0}, "h": {-1, 0}, "q": {-1, 0},
		"right": {1, 0}, "l": {1, 0}, "d": {1, 0},
	}
	for k, want := range cases {
		dx, dy, ok := dirFromKey(k)
		if !ok || dx != want[0] || dy != want[1] {
			t.Errorf("dirFromKey(%q) = (%d,%d,%v), attendu (%d,%d)", k, dx, dy, ok, want[0], want[1])
		}
	}
	if _, _, ok := dirFromKey("x"); ok {
		t.Error("une touche inconnue ne doit pas donner de direction")
	}
}

func TestBar(t *testing.T) {
	if got := bar(5, 10, 10); got != "█████░░░░░" {
		t.Fatalf("bar(5,10,10) = %q", got)
	}
	if got := bar(0, 10, 4); got != "░░░░" {
		t.Fatalf("bar(0,10,4) = %q", got)
	}
	if got := bar(20, 10, 4); got != "████" {
		t.Fatalf("bar au-delà du max doit être pleine: %q", got)
	}
	if got := bar(3, 0, 4); got != "████" {
		t.Fatalf("max=0 ne doit pas diviser par zéro: %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("court", 10); got != "court" {
		t.Fatalf("truncate ne doit pas couper: %q", got)
	}
	if got := truncate("une chaîne vraiment trop longue", 10); got != "une chaîn…" {
		t.Fatalf("truncate incorrect: %q", got)
	}
}

// TestShopScreenOuvrePresDuMarchand vérifie que m ouvre le shop quand le
// joueur est adjacent au marchand (snapshot avec marchand à côté).
func TestShopScreenOuvrePresDuMarchand(t *testing.T) {
	m, c := newTestModel(42)
	m.screen = screenPlaying
	// Place un marchand adjacent au joueur via le state du serveur,
	// puis rafraîchit le snapshot du modèle pour refléter le changement.
	s := c.State()
	s.CurrentMap().Merchant = game.NewMerchant(game.NewRNG(1), s.Player.X+1, s.Player.Y, 2)
	m.snapshot = s.Snapshot()
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if m.screen != screenShop {
		t.Fatal("m près du marchand doit ouvrir l'écran shop")
	}
	if v := m.View(); !strings.Contains(v, "Marchand") {
		t.Fatal("l'écran marchand doit s'afficher")
	}
}

// TestNewGameMsgFermeAncienClient — RÉGRESSION (17/10 16:44) : newGameMsg
// doit fermer l'ancien client pour éviter les joueurs fantômes côté serveur
// (quand on reconnecte via le menu après un --connect, ou qu'on relance).
func TestNewGameMsgFermeAncienClient(t *testing.T) {
	snap := game.NewState(42).Snapshot()
	old := &mockClient{snap: snap}
	m := New(old, snap, false)
	newC := &mockClient{snap: snap}
	updated, _ := m.Update(newGameMsg{client: newC, snapshot: snap})
	m = updated.(*Model)
	if !old.closed {
		t.Fatal("newGameMsg doit fermer l'ancien client (évite les joueurs fantômes)")
	}
	if m.client != client.GameClient(newC) {
		t.Fatal("le nouveau client doit remplacer l'ancien")
	}
}

// TestNewDirectPlayVaEnJeu — RÉGRESSION (17/10 16:44) : New(..., true) démarre
// directement en jeu (screenPlaying), pas sur le menu. Utilisé par --connect.
func TestNewDirectPlayVaEnJeu(t *testing.T) {
	snap := game.NewState(42).Snapshot()
	m := New(&mockClient{snap: snap}, snap, true)
	if m.screen != screenPlaying {
		t.Fatalf("directPlay=true doit démarrer en jeu, got screen=%v", m.screen)
	}
	// Init doit lancer waitSnapshot (boucle de lecture active).
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init doit lancer waitSnapshot en screenPlaying (directPlay)")
	}
}

// TestNewSansDirectPlayVaAuMenu vérifie que New(..., false) démarre sur le menu.
func TestNewSansDirectPlayVaAuMenu(t *testing.T) {
	snap := game.NewState(42).Snapshot()
	m := New(&mockClient{snap: snap}, snap, false)
	if m.screen != screenMenu {
		t.Fatalf("directPlay=false doit démarrer sur le menu, got screen=%v", m.screen)
	}
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init ne doit rien lancer sur le menu")
	}
}
