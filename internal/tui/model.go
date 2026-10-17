package tui

import (
	"game-test/internal/game"

	tea "github.com/charmbracelet/bubbletea"
)

// screen identifie l'écran courant de l'interface.
type screen int

const (
	screenMenu screen = iota
	screenPlaying
	screenHelp
	screenInventory
	screenScores
	screenGameOver
	screenShop
)

// hudWidth est la largeur fixe du panneau HUD affiché à droite de la carte.
const hudWidth = 40

// Model est le modèle BubbleTea du Donjon Caféiné.
// Toute la logique de jeu vit dans game.State ; le modèle ne fait que
// traduire les touches en actions et rendre l'état.
// Le rendu plein écran passe par un viewport : une caméra centrée sur le
// joueur, recalculée à chaque resize et à chaque action du joueur.
type Model struct {
	state      *game.State
	screen     screen
	width      int
	height     int
	viewX      int // coin supérieur gauche du viewport (coordonnées monde)
	viewY      int
	viewW      int // dimensions du viewport en cellules
	viewH      int
	lastDX     int
	lastDY     int
	menuCursor int
	scoreSaved bool
	invCursor  int // curseur dans l'inventaire
	shopCursor int // curseur dans l'écran marchand
	shopTab    int // onglet marchand : 0 = achat, 1 = vente
}

// New crée le modèle sur une partie donnée (démarrage sur le menu).
// Les dimensions par défaut (80x24) sont remplacées par le premier
// WindowSizeMsg envoyé par BubbleTea.
func New(s *game.State) *Model {
	m := &Model{state: s, screen: screenMenu, lastDX: 1, width: 80, height: 24}
	m.updateViewport()
	return m
}

// Init ne lance aucune commande au démarrage.
func (m *Model) Init() tea.Cmd { return nil }

// Run lance la boucle BubbleTea en plein écran alternatif.
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// updateViewport recalcule la caméra : un rectangle centré sur le joueur,
// borné aux limites du monde et limité par l'espace disponible dans le
// terminal (moins le HUD à droite, moins le titre et le pied de page).
func (m *Model) updateViewport() {
	w := m.state.World
	// Espace disponible pour la carte.
	availW := m.width - hudWidth - 2 // -2 : marge entre la carte et le HUD
	availH := m.height - 2           // -2 : titre (haut) + pied de page (bas)
	if availW < 10 {
		availW = 10
	}
	if availH < 5 {
		availH = 5
	}
	// Le viewport ne dépasse pas les dimensions du monde.
	m.viewW = min(availW, w.Width)
	m.viewH = min(availH, w.Height)
	// Centrer sur le joueur.
	m.viewX = m.state.Player.X - m.viewW/2
	m.viewY = m.state.Player.Y - m.viewH/2
	// Borner dans le monde.
	if m.viewX < 0 {
		m.viewX = 0
	}
	if m.viewY < 0 {
		m.viewY = 0
	}
	if m.viewX+m.viewW > w.Width {
		m.viewX = w.Width - m.viewW
	}
	if m.viewY+m.viewH > w.Height {
		m.viewY = w.Height - m.viewH
	}
	if m.viewX < 0 {
		m.viewX = 0
	}
	if m.viewY < 0 {
		m.viewY = 0
	}
}
