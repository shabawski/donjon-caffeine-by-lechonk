package tui

import (
	"time"

	"game-test/internal/client"
	"game-test/internal/game"
	"game-test/internal/net"
	"game-test/internal/score"

	tea "github.com/charmbracelet/bubbletea"
)

// snapshotMsg est reçu quand le serveur envoie un nouvel état du jeu.
type snapshotMsg struct{ snap *game.Snapshot }

// newGameMsg est reçu quand une nouvelle partie démarre (menu ou rejouer).
type newGameMsg struct {
	client   client.GameClient
	snapshot *game.Snapshot
}

// errMsg transporte une erreur du client (connexion, lecture).
type errMsg struct{ err error }

// connectErrMsg signale un échec de connexion TCP (affiché sous le champ).
type connectErrMsg struct{ err error }

// Update dispatch selon le type de message. Déterministe et sans effet de
// bord : les transitions d'état dépendent uniquement du message et de l'état.
// Les actions du joueur envoient un input au serveur et retournent une
// commande qui attend le prochain snapshot (bloquant côté tea.Cmd).
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateViewport()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case snapshotMsg:
		m.snapshot = msg.snap
		m.pendingInput = false
		m.afterSnapshot()
		// Relancer l'attente du prochain snapshot (UN SEUL waitSnapshot en
		// cours) : en multi, le serveur broadcast à tous les clients après
		// chaque input, donc on reçoit des snapshots même sans input (pour
		// voir les autres joueurs). En solo, la commande bloque jusqu'au
		// prochain input.
		if m.screen == screenPlaying {
			return m, m.waitSnapshot()
		}
		return m, nil
	case newGameMsg:
		// Ferme l'ancien client (évite les joueurs fantômes côté serveur quand
		// on reconnecte via le menu après un --connect, ou qu'on relance).
		if m.client != nil {
			m.client.Close()
		}
		m.client = msg.client
		m.snapshot = msg.snapshot
		m.screen = screenPlaying
		m.scoreSaved = false
		m.pendingInput = false
		m.lastDX, m.lastDY = 1, 0
		m.invCursor, m.shopCursor, m.shopTab = 0, 0, 0
		m.updateViewport()
		return m, m.waitSnapshot() // démarre la boucle de lecture
	case errMsg:
		return m, tea.Quit
	case connectErrMsg:
		m.connectErr = msg.err.Error()
		return m, nil
	}
	return m, nil
}

// sendInput envoie une action au serveur. NE RETOURNE PAS de waitSnapshot :
// le waitSnapshot en cours (lancé au démarrage ou après le dernier snapshot)
// recevra le prochain snapshot. En retourner un second en créerait deux en
// parallèle — le "mauvais" volerait le snapshot de réponse et le joueur
// semblerait bloqué (bug 17/10 15:20).
func (m *Model) sendInput(in net.InputPayload) (tea.Model, tea.Cmd) {
	m.client.DrainSnapshots()
	_ = m.client.SendInput(in)
	m.pendingInput = true
	return m, nil
}

// enterPlaying revient à l'écran de jeu depuis un overlay : draine les
// snapshots en attente (broadcasts reçus pendant l'overlay) et relance la
// boucle de lecture.
func (m *Model) enterPlaying() (tea.Model, tea.Cmd) {
	m.screen = screenPlaying
	m.client.DrainSnapshots()
	return m, m.waitSnapshot()
}

// waitSnapshot bloque jusqu'au prochain snapshot du serveur (tea.Cmd).
func (m *Model) waitSnapshot() tea.Cmd {
	return func() tea.Msg {
		snap, err := m.client.NextSnapshot()
		if err != nil {
			return errMsg{err}
		}
		if snap == nil {
			return nil // fin de partie : le dernier snapshot a déjà été reçu
		}
		return snapshotMsg{snap}
	}
}

// startNewGame crée une nouvelle partie solo (nouvelle graine) et retourne
// une commande qui construit le client local et attend le snapshot initial.
func (m *Model) startNewGame() tea.Cmd {
	return func() tea.Msg {
		c := client.NewLocalClient(time.Now().UnixNano(), game.DefaultWidth, game.DefaultHeight)
		snap, _ := c.NextSnapshot()
		return newGameMsg{client: c, snapshot: snap}
	}
}

// afterSnapshot est appelé à chaque snapshot reçu : recalcule le viewport et
// détecte la fin de partie (sauvegarde le score une fois, bascule sur l'écran
// de fin).
func (m *Model) afterSnapshot() {
	m.updateViewport()
	if m.snapshot.Phase != game.PhasePlaying && !m.scoreSaved {
		m.scoreSaved = true
		m.screen = screenGameOver
		_ = score.Add(entryFromSnapshot(m.snapshot))
	}
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
			return m.enterPlaying()
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
	case screenConnect:
		return m.updateConnect(msg)
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
		if m.menuCursor < 3 {
			m.menuCursor++
		}
	case "enter":
		switch m.menuCursor {
		case 0:
			return m, m.startNewGame() // nouvelle partie solo
		case 1:
			// Multijoueur : écran de saisie de l'adresse serveur.
			m.screen = screenConnect
			m.connectInput = ""
			m.connectErr = ""
		case 2:
			m.screen = screenScores
		case 3:
			return m, tea.Quit
		}
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// updateConnect gère la saisie de l'adresse serveur (host:port) et la connexion.
func (m *Model) updateConnect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		addr := m.connectInput
		if addr == "" {
			return m, nil
		}
		return m, m.connectCmd(addr)
	case tea.KeyEsc:
		m.screen = screenMenu
	case tea.KeyBackspace:
		if n := len(m.connectInput); n > 0 {
			m.connectInput = m.connectInput[:n-1]
		}
	case tea.KeySpace:
		m.connectInput += " "
	case tea.KeyRunes:
		m.connectInput += string(msg.Runes)
	}
	return m, nil
}

// connectCmd tente une connexion TCP au serveur distant (bloquant dans tea.Cmd).
// En cas de succès, produit un newGameMsg avec le client TCP et le snapshot initial.
func (m *Model) connectCmd(addr string) tea.Cmd {
	return func() tea.Msg {
		c, err := client.NewTCPClient(addr)
		if err != nil {
			return connectErrMsg{err}
		}
		if err := c.Connect(); err != nil {
			return connectErrMsg{err}
		}
		snap, err := c.NextSnapshot()
		if err != nil || snap == nil {
			c.Close()
			return connectErrMsg{err}
		}
		return newGameMsg{client: c, snapshot: snap}
	}
}

func (m *Model) updatePlaying(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if dx, dy, ok := dirFromKey(k); ok {
		m.lastDX, m.lastDY = dx, dy
		return m.sendInput(net.InputPayload{Kind: net.InputMove, DX: dx, DY: dy})
	}
	switch k {
	case ".":
		return m.sendInput(net.InputPayload{Kind: net.InputWait})
	case "f":
		return m.sendInput(net.InputPayload{Kind: net.InputFireball, LastDX: m.lastDX, LastDY: m.lastDY})
	case "g", ">":
		return m.sendInput(net.InputPayload{Kind: net.InputDescend})
	case "G", "<":
		return m.sendInput(net.InputPayload{Kind: net.InputAscend})
	case "i":
		m.screen = screenInventory
		m.invCursor = 0
		m.client.DrainSnapshots()
	case "m":
		// Ouvrir le marchand si adjacent (distance de Chebyshev <= 1).
		if m.snapshot.Merchant != nil && chebyshev(
			m.snapshot.Player.X, m.snapshot.Player.Y,
			m.snapshot.Merchant.X, m.snapshot.Merchant.Y) <= 1 {
			m.screen = screenShop
			m.shopCursor = 0
			m.shopTab = 0
			m.client.DrainSnapshots()
		}
	case "?":
		m.screen = screenHelp
		m.client.DrainSnapshots()
	case "Q", "esc":
		m.screen = screenMenu // abandon de la partie en cours (q = gauche en ZQSD)
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// updateInventory gère les touches dans l'écran inventaire :
// navigation, équiper (e), utiliser (u), jeter (d).
// Chaque action envoie un input au serveur et attend le snapshot résultant.
func (m *Model) updateInventory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inv := m.snapshot.Player.Inventory
	switch msg.String() {
	case "esc", "i", "enter":
		return m.enterPlaying()
	case "up", "k", "z":
		if m.invCursor > 0 {
			m.invCursor--
		}
	case "down", "j", "s":
		if m.invCursor < len(inv)-1 {
			m.invCursor++
		}
	case "e":
		return m.sendInput(net.InputPayload{Kind: net.InputEquip, Index: m.invCursor})
	case "u":
		return m.sendInput(net.InputPayload{Kind: net.InputUse, Index: m.invCursor})
	case "d":
		return m.sendInput(net.InputPayload{Kind: net.InputDrop, Index: m.invCursor})
	}
	return m, nil
}

// updateShop gère les touches dans l'écran marchand :
// tab change d'onglet (achat/vente), entrée achète ou vend.
func (m *Model) updateShop(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "m":
		return m.enterPlaying()
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
			return m.sendInput(net.InputPayload{Kind: net.InputBuy, Index: m.shopCursor})
		}
		return m.sendInput(net.InputPayload{Kind: net.InputSell, Index: m.shopCursor})
	}
	return m, nil
}

// shopListLen renvoie le nombre d'entrées dans l'onglet actif du marchand.
func (m *Model) shopListLen() int {
	if m.shopTab == 0 {
		if m.snapshot.Merchant == nil {
			return 0
		}
		return len(m.snapshot.Merchant.Shop)
	}
	return len(m.snapshot.Player.Inventory)
}

func (m *Model) updateGameOver(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		// Nouvelle partie solo avec une graine aléatoire.
		return m, m.startNewGame()
	case "s":
		m.screen = screenScores
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// chebyshev est la distance de Chebyshev (max des distances sur chaque axe).
func chebyshev(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// entryFromSnapshot convertit le snapshot final en entrée de high score.
func entryFromSnapshot(s *game.Snapshot) score.Entry {
	return score.Entry{
		Seed:  s.Seed,
		Depth: s.Depth,
		Level: s.Player.Level,
		Gold:  s.Player.Gold,
		Kills: s.Player.Kills,
		Turns: s.Player.Turns,
		Score: s.Score(),
		Won:   s.Won,
		Death: s.CauseOfDeath,
	}
}
