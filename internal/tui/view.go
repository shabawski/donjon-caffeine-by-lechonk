package tui

import (
	"fmt"
	"strings"

	"game-test/internal/game"

	"github.com/charmbracelet/lipgloss"
)

// View rend l'écran courant. Ne mutue jamais l'état.
func (m *Model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenPlaying:
		return m.viewGame()
	case screenHelp:
		return m.viewHelp()
	case screenInventory:
		return m.viewInventory()
	case screenScores:
		return m.viewScores()
	case screenGameOver:
		return m.viewGameOver()
	}
	return ""
}

// viewGame rend le titre, la carte (viewport centré sur le joueur) à gauche,
// le HUD à droite, et un pied de page avec les raccourcis clavier.
func (m *Model) viewGame() string {
	title := styleTitle.Render(fmt.Sprintf("☕ Donjon Caféiné — étage %d/%d", m.state.World.Depth, game.MaxDepth))
	mapW := m.width - hudWidth - 2
	if mapW < m.viewW {
		mapW = m.viewW
	}
	mapBlock := lipgloss.NewStyle().Width(mapW).Render(m.renderViewport())
	body := lipgloss.JoinHorizontal(lipgloss.Top, mapBlock, m.renderHUD())
	footer := styleDim.Render("ZQSD/HJKL/flèches: bouger · f: feu · g: escalier · i: inventaire · ?: aide · Q: quitter")
	return title + "\n" + body + "\n" + footer
}

// renderViewport dessine la portion du monde visible dans le viewport,
// centrée sur le joueur. Fog of war : cases jamais vues en espace,
// cases explorées mais hors vue en atténué, cases visibles en couleur.
func (m *Model) renderViewport() string {
	s := m.state
	var b strings.Builder
	for vy := 0; vy < m.viewH; vy++ {
		wy := m.viewY + vy
		for vx := 0; vx < m.viewW; vx++ {
			wx := m.viewX + vx
			if !s.World.InBounds(wx, wy) {
				b.WriteString(" ")
				continue
			}
			i := wy*s.World.Width + wx
			switch {
			case !s.FOV.Explored[i]:
				b.WriteString(" ")
			case !s.FOV.Visible[i]:
				b.WriteString(styleDim.Render(string(s.World.Tiles[i].Glyph())))
			default:
				r, st := glyphAt(s, wx, wy)
				b.WriteString(st.Render(string(r)))
			}
		}
		if vy < m.viewH-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// glyphAt choisit le glyph et le style d'une case visible :
// joueur > monstre > objet > tuile.
func glyphAt(s *game.State, x, y int) (rune, lipgloss.Style) {
	if s.Player.X == x && s.Player.Y == y {
		return s.Player.Glyph, stylePlayer
	}
	if e := s.OccupiedByEnemy(x, y); e != nil {
		return e.Glyph, styleForKind(e.Kind)
	}
	if it := s.ItemAt(x, y); it != nil {
		return it.Glyph, styleForItem(it.Kind)
	}
	t := s.World.Tiles[y*s.World.Width+x]
	switch t {
	case game.TileStairs:
		return t.Glyph(), styleStairs
	case game.TileWall:
		return t.Glyph(), styleWall
	default:
		return t.Glyph(), styleFloor
	}
}

// renderHUD rend la colonne de droite : statistiques, équipement, journal.
// Le journal occupe tout l'espace vertical restant dans le viewport.
// Le rendu produit exactement viewH lignes (sans saut de ligne final) pour
// que JoinHorizontal aligne correctement la carte et le HUD.
func (m *Model) renderHUD() string {
	s := m.state
	p := &s.Player
	lines := make([]string, 0, m.viewH)
	lines = append(lines, styleHP.Render(bar(p.HP, p.MaxHP, 12))+fmt.Sprintf(" PV %d/%d", p.HP, p.MaxHP))
	lines = append(lines, styleMana.Render(bar(p.Mana, p.MaxMana, 12))+fmt.Sprintf(" Mana %d/%d", p.Mana, p.MaxMana))
	lines = append(lines, fmt.Sprintf("Niv %d  XP %d/%d", p.Level, p.XP, game.XPNeededForLevel(p.Level+1)))
	lines = append(lines, fmt.Sprintf("ATK %d  DEF %d", p.EffectiveAttack(), p.EffectiveDefense()))
	lines = append(lines, fmt.Sprintf("Or %d   Kills %d   Tours %d", p.Gold, p.Kills, p.Turns))
	lines = append(lines, "Arme   "+equipName(p.WeaponName, p.WeaponBonus))
	lines = append(lines, "Armure "+equipName(p.ArmorName, p.ArmorBonus))
	lines = append(lines, "")
	lines = append(lines, "── Journal ──")
	// Le journal occupe tout l'espace vertical restant dans le viewport.
	const hudHeaderLines = 9 // stats (7) + ligne vide (1) + titre journal (1)
	logLines := m.viewH - hudHeaderLines
	if logLines < 3 {
		logLines = 3
	}
	start := len(s.Log) - logLines
	if start < 0 {
		start = 0
	}
	for _, line := range s.Log[start:] {
		lines = append(lines, styleLog.Render("> "+truncate(line, hudWidth-4)))
	}
	// Remplit les lignes restantes pour atteindre exactement viewH lignes.
	for len(lines) < hudHeaderLines+logLines {
		lines = append(lines, "")
	}
	return styleHUD.Render(strings.Join(lines, "\n"))
}

// equipName formate une ligne d'équipement (arme/armure).
func equipName(name string, bonus int) string {
	if name == "" {
		return "aucune (+0)"
	}
	return fmt.Sprintf("%s (+%d)", name, bonus)
}

// bar rend une barre de progression textuelle de la largeur donnée.
func bar(cur, max, width int) string {
	if max <= 0 {
		max = 1
	}
	filled := cur * width / max
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// truncate coupe une chaîne à n runes (avec ellipse).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
