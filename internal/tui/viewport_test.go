package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// playerInViewport vérifie que le joueur est dans le rectangle du viewport.
func playerInViewport(m *Model) bool {
	p := &m.snapshot.Player
	return p.X >= m.viewX && p.X < m.viewX+m.viewW &&
		p.Y >= m.viewY && p.Y < m.viewY+m.viewH
}

func TestViewportContainsPlayer(t *testing.T) {
	m, _ := newTestModel(42)
	m.width, m.height = 80, 24
	m.updateViewport()
	if !playerInViewport(m) {
		t.Fatalf("joueur (%d,%d) hors viewport [%d,%d[-[%d,%d[",
			m.snapshot.Player.X, m.snapshot.Player.Y, m.viewX, m.viewY, m.viewX+m.viewW, m.viewY+m.viewH)
	}
}

func TestViewportClampsToWorld(t *testing.T) {
	m, _ := newTestModel(42)
	// Terminal immense : le viewport ne dépasse pas le monde.
	m.width, m.height = 500, 200
	m.updateViewport()
	if m.viewW > m.snapshot.Width || m.viewH > m.snapshot.Height {
		t.Fatalf("viewport %dx%d dépasse le monde %dx%d",
			m.viewW, m.viewH, m.snapshot.Width, m.snapshot.Height)
	}
	if m.viewX < 0 || m.viewY < 0 ||
		m.viewX+m.viewW > m.snapshot.Width || m.viewY+m.viewH > m.snapshot.Height {
		t.Fatalf("viewport hors du monde: [%d,%d]-[%d,%d]",
			m.viewX, m.viewY, m.viewX+m.viewW, m.viewY+m.viewH)
	}
}

func TestViewportFollowsPlayerAtEdge(t *testing.T) {
	m, c := newTestModel(42)
	m.width, m.height = 60, 20
	m.updateViewport()
	// Téléporte le joueur dans le coin inférieur droit du monde (via le state).
	s := c.State()
	s.Player.X = s.World.Width - 2
	s.Player.Y = s.World.Height - 2
	// Force un snapshot pour refléter la nouvelle position.
	m.snapshot = s.Snapshot()
	m.updateViewport()
	if !playerInViewport(m) {
		t.Fatalf("joueur (%d,%d) hors viewport après téléportation [%d,%d[-[%d,%d[",
			m.snapshot.Player.X, m.snapshot.Player.Y, m.viewX, m.viewY, m.viewX+m.viewW, m.viewY+m.viewH)
	}
	if m.viewX+m.viewW > m.snapshot.Width || m.viewY+m.viewH > m.snapshot.Height {
		t.Fatal("le viewport dépasse le monde après téléportation")
	}
}

func TestWindowSizeMsgUpdatesViewport(t *testing.T) {
	m, _ := newTestModel(42)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 || m.height != 40 {
		t.Fatalf("dimensions terminal non mises à jour: %dx%d", m.width, m.height)
	}
	// availW = 120 - hudWidth(40) - 2 = 78 ; le monde fait 80 → viewW = 78.
	if m.viewW != 78 {
		t.Fatalf("viewW attendu 78, got %d", m.viewW)
	}
	// availH = 40 - 2 = 38 ; le monde fait 40 → viewH = 38.
	if m.viewH != 38 {
		t.Fatalf("viewH attendu 38, got %d", m.viewH)
	}
	if !playerInViewport(m) {
		t.Fatal("le joueur doit rester visible après le resize")
	}
}

func TestViewportAfterPlayerMove(t *testing.T) {
	m, c := newTestModel(42)
	m.screen = screenPlaying
	// Place le joueur au milieu d'une zone ouverte (via le state du serveur).
	s := c.State()
	found := false
outer:
	for y := 1; y < s.World.Height-1 && !found; y++ {
		for x := 1; x < s.World.Width-1 && !found; x++ {
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
	m.width, m.height = 60, 20
	m.snapshot = s.Snapshot()
	m.updateViewport()
	m = press(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !playerInViewport(m) {
		t.Fatal("le joueur doit rester visible après un déplacement")
	}
}

func TestViewGameFillsTerminalHeight(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenPlaying
	m.width, m.height = 100, 30
	m.updateViewport()
	v := m.View()
	lines := strings.Count(v, "\n") + 1
	// titre (1) + viewport (viewH) + footer (1) = height.
	if lines != m.height {
		t.Fatalf("la vue doit faire %d lignes (hauteur terminal), got %d", m.height, lines)
	}
}

func TestViewGameContainsFooter(t *testing.T) {
	m, _ := newTestModel(42)
	m.screen = screenPlaying
	v := m.View()
	if !strings.Contains(v, "ZQSD") {
		t.Fatal("le pied de page avec les raccourcis doit être rendu")
	}
}
