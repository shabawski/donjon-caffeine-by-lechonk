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
		m.updateViewport()
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
	case screenHelp:
		switch msg.String() {
		case "esc", "?", "enter":
			m.screen = screenPlaying
		}
		return m, nil
	case screenInventory:
		return m.updateInventory(msg)
	case screenShop:
		return m.updateShop(msg)
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
	case "G", "<":
		m.state.Ascend()
		m.afterPlayerAction()
	case "i":
		m.screen = screenInventory
		m.invCursor = 0
	case "m":
		if m.state.MerchantAdjacent() {
			m.screen = screenShop
			m.shopCursor = 0
			m.shopTab = 0
		}
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
// et bascule sur l'écran de fin. Recalcule aussi le viewport pour que la
// caméra suive le joueur.
func (m *Model) afterPlayerAction() {
	m.updateViewport()
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
		m.updateViewport()
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

// updateInventory gère les touches dans l'écran inventaire :
// navigation, équiper (e), utiliser (u), jeter (d).
func (m *Model) updateInventory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := m.state.Player.Inventory
	switch msg.String() {
	case "esc", "i", "enter":
		m.screen = screenPlaying
	case "up", "k", "z":
		if m.invCursor > 0 {
			m.invCursor--
		}
	case "down", "j", "s":
		if m.invCursor < len(inv)-1 {
			m.invCursor++
		}
	case "e":
		m.state.EquipItem(m.invCursor)
		m.clampInvCursor()
	case "u":
		m.state.UseItem(m.invCursor)
		m.clampInvCursor()
	case "d":
		m.state.DropItem(m.invCursor)
		m.clampInvCursor()
	}
	return m, nil
}

// clampInvCursor borne le curseur d'inventaire après une suppression.
func (m *Model) clampInvCursor() {
	if n := len(m.state.Player.Inventory); m.invCursor >= n && m.invCursor > 0 {
		m.invCursor = n - 1
	}
}

// updateShop gère les touches dans l'écran marchand :
// tab change d'onglet (achat/vente), entrée achète ou vend.
func (m *Model) updateShop(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "m":
		m.screen = screenPlaying
	case "tab", "h", "l":
		m.shopTab = 1 - m.shopTab // bascule achat (0) / vente (1)
		m.shopCursor = 0
	case "up", "k", "z":
		if m.shopCursor > 0 {
			m.shopCursor--
		}
	case "down", "j", "s":
		if max := m.shopListLen(); m.shopCursor < max-1 {
			m.shopCursor++
		}
	case "enter":
		if m.shopTab == 0 {
			m.state.BuyItem(m.shopCursor)
		} else {
			m.state.SellItem(m.shopCursor)
		}
		if n := m.shopListLen(); m.shopCursor >= n && m.shopCursor > 0 {
			m.shopCursor = n - 1
		}
	}
	return m, nil
}

// shopListLen renvoie le nombre d'entrées dans l'onglet actif du marchand.
func (m *Model) shopListLen() int {
	if m.shopTab == 0 {
		if m.state.World.Merchant == nil {
			return 0
		}
		return len(m.state.World.Merchant.Shop)
	}
	return len(m.state.Player.Inventory)
}
