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
)

// Model est le modèle BubbleTea du Donjon Caféiné.
// Toute la logique de jeu vit dans game.State ; le modèle ne fait que
// traduire les touches en actions et rendre l'état.
type Model struct {
	state      *game.State
	screen     screen
	width      int
	height     int
	lastDX     int
	lastDY     int
	menuCursor int
	scoreSaved bool
}

// New crée le modèle sur une partie donnée (démarrage sur le menu).
func New(s *game.State) *Model {
	return &Model{state: s, screen: screenMenu, lastDX: 1}
}

// Init ne lance aucune commande au démarrage.
func (m *Model) Init() tea.Cmd { return nil }

// Run lance la boucle BubbleTea en plein écran alternatif.
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
