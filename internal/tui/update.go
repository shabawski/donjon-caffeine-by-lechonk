package tui

import (
	"time"

	"game-test/internal/game"
	"game-test/internal/score"

	tea "github.com/charmbracelet/bubbletea"
)

// Update dispatch selon le type de message. Déterministe et sans effet de
// bord : les transitions d'état dépendent uniquement du message et de l'état.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenPlaying:
		return m.updatePlaying(msg)
	case screenHelp, screenInventory:
		switch msg.String() {
		case "esc", "?", "i", "enter":
			m.screen = screenPlaying
		}
		return m, nil
	case screenScores:
		switch msg.String() {
		case "esc", "q", "enter":
			m.screen = screenMenu
		}
		return m, nil
	case screenGameOver:
		return m.updateGameOver(msg)
	}
	return m, nil
}

func (m *Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "z":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j", "s":
		if m.menuCursor < 2 {
			m.menuCursor++
		}
	case "enter":
		switch m.menuCursor {
		case 0:
			m.screen = screenPlaying
		case 1:
			m.screen = screenScores
		case 2:
			return m, tea.Quit
		}
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) updatePlaying(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if dx, dy, ok := dirFromKey(k); ok {
		m.lastDX, m.lastDY = dx, dy
		m.state.MovePlayer(dx, dy)
		m.afterPlayerAction()
		return m, nil
	}
	switch k {
	case ".":
		m.state.PlayerWait()
		m.afterPlayerAction()
	case "f":
		m.state.CastFireball(m.lastDX, m.lastDY)
		m.afterPlayerAction()
	case "g", ">":
		m.state.Descend()
		m.afterPlayerAction()
	case "i":
		m.screen = screenInventory
	case "?":
		m.screen = screenHelp
	case "Q", "esc":
		m.screen = screenMenu // abandon de la partie en cours (q = gauche en ZQSD)
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// afterPlayerAction détecte la fin de partie : sauvegarde le score une fois
// et bascule sur l'écran de fin.
func (m *Model) afterPlayerAction() {
	if m.state.Phase != game.PhasePlaying && !m.scoreSaved {
		m.scoreSaved = true
		m.screen = screenGameOver
		_ = score.Add(entryFromState(m.state))
	}
}

func (m *Model) updateGameOver(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		// Nouvelle partie avec une graine aléatoire.
		m.state = game.NewState(time.Now().UnixNano())
		m.screen = screenPlaying
		m.scoreSaved = false
		m.lastDX, m.lastDY = 1, 0
	case "s":
		m.screen = screenScores
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// entryFromState convertit l'état final de la partie en entrée de high score.
func entryFromState(s *game.State) score.Entry {
	return score.Entry{
		Seed:  s.Seed,
		Depth: s.World.Depth,
		Level: s.Player.Level,
		Gold:  s.Player.Gold,
		Kills: s.Player.Kills,
		Turns: s.Player.Turns,
		Score: s.Score(),
		Won:   s.Won,
		Death: s.CauseOfDeath,
	}
}
