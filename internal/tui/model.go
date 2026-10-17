package tui

import (
	"game-test/internal/client"
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
	screenConnect
)

// hudWidth est la largeur fixe du panneau HUD affiché à droite de la carte.
const hudWidth = 40

// Model est le modèle BubbleTea du Donjon Caféiné.
// Architecture client/serveur : le TUI ne touche plus au game.State.
// Il envoie des inputs via client.GameClient et rend depuis le dernier
// *game.Snapshot reçu du serveur (local via channels en solo, TCP en multi).
// Le rendu plein écran passe par un viewport : une caméra centrée sur le
// joueur, recalculée à chaque resize et à chaque snapshot reçu.
type Model struct {
	client       client.GameClient // solo (channels) ou multi (TCP)
	snapshot     *game.Snapshot    // dernier état reçu du serveur
	screen       screen
	width        int
	height       int
	viewX        int // coin supérieur gauche du viewport (coordonnées monde)
	viewY        int
	viewW        int // dimensions du viewport en cellules
	viewH        int
	lastDX       int
	lastDY       int
	menuCursor   int
	mode         game.GameMode // mode de la partie en cours (COOP ou MMORPG)
	scoreSaved   bool
	invCursor    int    // curseur dans l'inventaire
	shopCursor   int    // curseur dans l'écran marchand
	shopTab      int    // onglet marchand : 0 = achat, 1 = vente
	connectInput string // adresse host:port en cours de saisie (écran connexion)
	connectErr   string // dernière erreur de connexion (affichée sous le champ)
	pendingInput bool   // un input a été envoyé, le snapshot de réponse n'est pas encore reçu (tests)
}

// New crée le modèle sur un client connecté. Si directPlay est true (ex:
// --connect host:port), le modèle démarre directement en jeu (screenPlaying) ;
// sinon il démarre sur le menu (solo : l'utilisateur choisit solo/multi/scores).
// Les dimensions par défaut (80x24) sont remplacées par le premier
// WindowSizeMsg envoyé par BubbleTea.
func New(c client.GameClient, snap *game.Snapshot, directPlay bool) *Model {
	m := &Model{client: c, snapshot: snap, lastDX: 1, width: 80, height: 24, mode: game.ModeCoop}
	if directPlay {
		m.screen = screenPlaying
	} else {
		m.screen = screenMenu
	}
	m.updateViewport()
	return m
}

// Init lance la boucle de lecture des snapshots si on démarre directement en
// jeu (--connect). Sur le menu, aucune commande n'est lancée.
func (m *Model) Init() tea.Cmd {
	if m.screen == screenPlaying {
		return m.waitSnapshot()
	}
	return nil
}

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
	w := m.snapshot.Width
	h := m.snapshot.Height
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
	m.viewW = min(availW, w)
	m.viewH = min(availH, h)
	// Centrer sur le joueur.
	m.viewX = m.snapshot.Player.X - m.viewW/2
	m.viewY = m.snapshot.Player.Y - m.viewH/2
	// Borner dans le monde.
	if m.viewX < 0 {
		m.viewX = 0
	}
	if m.viewY < 0 {
		m.viewY = 0
	}
	if m.viewX+m.viewW > w {
		m.viewX = w - m.viewW
	}
	if m.viewY+m.viewH > h {
		m.viewY = h - m.viewH
	}
	if m.viewX < 0 {
		m.viewX = 0
	}
	if m.viewY < 0 {
		m.viewY = 0
	}
}
