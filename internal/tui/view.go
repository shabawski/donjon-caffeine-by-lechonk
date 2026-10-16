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

// viewGame rend le titre, la carte à gauche et le HUD/journal à droite.
func (m *Model) viewGame() string {
	title := styleTitle.Render(fmt.Sprintf("☕ Donjon Caféiné — étage %d/%d", m.state.World.Depth, game.MaxDepth))
	body := lipgloss.JoinHorizontal(lipgloss.Top, renderMap(m.state), m.renderHUD())
	return title + "\n" + body
}

// renderMap dessine la carte avec fog of war : cases jamais vues en espace,
// cases explorées mais hors vue en atténué, cases visibles en couleur.
func renderMap(s *game.State) string {
	var b strings.Builder
	for y := 0; y < s.World.Height; y++ {
		for x := 0; x < s.World.Width; x++ {
			i := y*s.World.Width + x
			switch {
			case !s.FOV.Explored[i]:
				b.WriteString(" ")
			case !s.FOV.Visible[i]:
				b.WriteString(styleDim.Render(string(s.World.Tiles[i].Glyph())))
			default:
				r, st := glyphAt(s, x, y)
				b.WriteString(st.Render(string(r)))
			}
		}
		if y < s.World.Height-1 {
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
func (m *Model) renderHUD() string {
	s := m.state
	p := &s.Player
	var b strings.Builder
	b.WriteString(styleHP.Render(bar(p.HP, p.MaxHP, 12)) + fmt.Sprintf(" PV %d/%d\n", p.HP, p.MaxHP))
	b.WriteString(styleMana.Render(bar(p.Mana, p.MaxMana, 12)) + fmt.Sprintf(" Mana %d/%d\n", p.Mana, p.MaxMana))
	b.WriteString(fmt.Sprintf("Niv %d  XP %d/%d\n", p.Level, p.XP, game.XPNeededForLevel(p.Level+1)))
	b.WriteString(fmt.Sprintf("ATK %d  DEF %d\n", p.EffectiveAttack(), p.EffectiveDefense()))
	b.WriteString(fmt.Sprintf("Or %d   Kills %d   Tours %d\n", p.Gold, p.Kills, p.Turns))
	b.WriteString("Arme   " + equipName(p.WeaponName, p.WeaponBonus) + "\n")
	b.WriteString("Armure " + equipName(p.ArmorName, p.ArmorBonus) + "\n")
	b.WriteString("\n── Journal ──\n")
	const logLines = 8
	start := len(s.Log) - logLines
	if start < 0 {
		start = 0
	}
	for _, line := range s.Log[start:] {
		b.WriteString(styleLog.Render("> "+truncate(line, 36)) + "\n")
	}
	return styleHUD.Render(b.String())
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
