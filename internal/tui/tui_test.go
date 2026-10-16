package tui

import (
	"strings"
	"testing"

	"game-test/internal/game"
	"game-test/internal/score"

	tea "github.com/charmbracelet/bubbletea"
)

func TestViewMenuContainsTitle(t *testing.T) {
	m := New(game.NewState(42))
	if v := m.View(); !strings.Contains(v, "DONJON") {
		t.Fatalf("titre du menu absent: %q", v)
	}
}

func TestMenuEnterStartsGame(t *testing.T) {
	m := New(game.NewState(42))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenPlaying {
		t.Fatal("enter doit lancer la partie depuis le menu")
	}
}

func TestMenuQuit(t *testing.T) {
	m := New(game.NewState(42))
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q doit produire tea.Quit depuis le menu")
	}
}

func TestMenuNavigationWraps(t *testing.T) {
	m := New(game.NewState(42))
	m.Update(tea.KeyMsg{Type: tea.KeyUp}) // remonte au-dessus de 0 : reste à 0
	if m.menuCursor != 0 {
		t.Fatalf("curseur attendu 0, got %d", m.menuCursor)
	}
	for i := 0; i < 5; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.menuCursor != 2 {
		t.Fatalf("curseur plafonné à 2, got %d", m.menuCursor)
	}
}

func TestPlayingMoveKey(t *testing.T) {
	s := game.NewState(42)
	// Trouve une case avec la case de droite libre.
	found := false
outer:
	for y := 0; y < s.World.Height; y++ {
		for x := 0; x+1 < s.World.Width; x++ {
			if s.World.IsWalkable(x, y) && s.World.IsWalkable(x+1, y) {
				s.Player.X, s.Player.Y = x, y
				found = true
				break outer
			}
		}
	}
	if !found {
		t.Fatal("pas de case libre trouvée")
	}
	m := New(s)
	m.screen = screenPlaying
	x0 := s.Player.X
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if s.Player.X != x0+1 {
		t.Fatalf("la touche l doit déplacer à droite: %d -> %d", x0, s.Player.X)
	}
}

func TestViewGameContainsPlayerGlyphAndHUD(t *testing.T) {
	m := New(game.NewState(42))
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
	m := New(game.NewState(42))
	m.screen = screenPlaying
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.screen != screenHelp {
		t.Fatal("? doit ouvrir l'aide")
	}
	if v := m.View(); !strings.Contains(v, "boule de feu") {
		t.Fatal("l'aide doit lister les commandes")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenPlaying {
		t.Fatal("esc doit fermer l'aide")
	}
}

func TestInventoryOverlayRoundTrip(t *testing.T) {
	m := New(game.NewState(42))
	m.screen = screenPlaying
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.screen != screenInventory {
		t.Fatal("i doit ouvrir l'inventaire")
	}
	if v := m.View(); !strings.Contains(v, "Inventaire") {
		t.Fatal("l'inventaire doit s'afficher")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.screen != screenPlaying {
		t.Fatal("i doit fermer l'inventaire")
	}
}

func TestQuitFromGameReturnsToMenu(t *testing.T) {
	m := New(game.NewState(42))
	m.screen = screenPlaying
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenMenu {
		t.Fatal("esc en jeu doit ramener au menu")
	}
	// Q majuscule fait de même.
	m.screen = screenPlaying
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	if m.screen != screenMenu {
		t.Fatal("Q en jeu doit ramener au menu")
	}
}

func TestZQSDQMovesLeft(t *testing.T) {
	s := game.NewState(42)
	// Trouve une case avec la case de gauche libre.
	found := false
outer:
	for y := 0; y < s.World.Height; y++ {
		for x := 1; x < s.World.Width; x++ {
			if s.World.IsWalkable(x, y) && s.World.IsWalkable(x-1, y) {
				s.Player.X, s.Player.Y = x, y
				found = true
				break outer
			}
		}
	}
	if !found {
		t.Fatal("pas de case libre trouvée")
	}
	m := New(s)
	m.screen = screenPlaying
	x0 := s.Player.X
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if s.Player.X != x0-1 {
		t.Fatalf("la touche q (ZQSD) doit déplacer à gauche: %d -> %d", x0, s.Player.X)
	}
}

func TestDeathTriggersGameOverAndSavesScore(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := game.NewState(42)
	m := New(s)
	m.screen = screenPlaying
	s.Player.HP = 0
	s.PlayerDeath()
	m.afterPlayerAction()
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
	m := New(s)
	m.screen = screenPlaying
	// Simule une victoire via le moteur.
	s.Player.HP = 0 //neutre
	s.Player.HP = s.Player.MaxHP
	boss := s.Enemies[0]
	boss.Kind = game.KindBoss
	boss.HP = 1
	s.Player.Attack = 1000
	s.Player.X, s.Player.Y = boss.X, boss.Y
	s.MovePlayer(0, 0)
	m.afterPlayerAction()
	if m.screen != screenGameOver {
		t.Fatal("écran de fin attendu après la victoire")
	}
	if v := m.View(); !strings.Contains(v, "VICTOIRE") {
		t.Fatalf("l'écran de fin doit annoncer la victoire: %q", v)
	}
}

func TestGameOverRejouerStartsNewGame(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := game.NewState(42)
	m := New(s)
	m.screen = screenGameOver
	m.scoreSaved = true
	oldSeed := s.Seed
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.screen != screenPlaying {
		t.Fatal("r doit relancer une partie")
	}
	if m.scoreSaved {
		t.Fatal("le flag scoreSaved doit être remis à zéro")
	}
	if m.state.Seed == oldSeed {
		t.Fatal("une nouvelle partie doit avoir une nouvelle graine")
	}
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
