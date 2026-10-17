package tui

import (
	"fmt"
	"strings"

	"game-test/internal/game"
)

// viewHelp rend l'aide des commandes.
func (m *Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("❓ Aide — commandes") + "\n\n")
	rows := [][2]string{
		{"ZQSD / HJKL / flèches", "se déplacer (attaquer si un monstre est adjacent)"},
		{". (point)", "attendre un tour"},
		{"f", "boule de feu (8 mana, dégâts de zone)"},
		{"g ou >", "descendre l'escalier (se tenir dessus)"},
		{"G ou <", "remonter l'escalier (se tenir dessus)"},
		{"e", "entrer dans un bâtiment ⌂ (MMORPG, monde ouvert)"},
		{"x", "sortir du donjon vers le monde ouvert (MMORPG)"},
		{"i", "inventaire (équiper, utiliser, jeter)"},
		{"m", "marchand (se tenir près du PNJ M)"},
		{"?", "cette aide"},
		{"Q (maj) ou esc", "retour au menu (abandon)"},
		{"Objectif", "tuer le Seigneur des Tasses au 10e étage"},
	}
	for _, r := range rows {
		b.WriteString(styleCursor.Render(fmt.Sprintf("  %-22s", r[0])) + r[1] + "\n")
	}
	b.WriteString("\n" + styleDim.Render("esc/?/entrée : retour au jeu"))
	return b.String()
}

// viewInventory rend l'inventaire : équipement actuel + liste des objets
// gardés sur soi, avec curseur de sélection (e équiper, u utiliser, d jeter).
func (m *Model) viewInventory() string {
	p := &m.snapshot.Player
	var b strings.Builder
	b.WriteString(styleTitle.Render("🎒 Inventaire") + "\n\n")
	b.WriteString(fmt.Sprintf("  Arme   : %s\n", equipName(p.WeaponName, p.WeaponBonus)))
	b.WriteString(fmt.Sprintf("  Armure : %s\n", equipName(p.ArmorName, p.ArmorBonus)))
	b.WriteString(fmt.Sprintf("  Or     : %d pièces\n", p.Gold))
	b.WriteString(fmt.Sprintf("  PV     : %d/%d\n", p.HP, p.MaxHP))
	b.WriteString(fmt.Sprintf("  Mana   : %d/%d\n", p.Mana, p.MaxMana))
	b.WriteString(fmt.Sprintf("  ATK    : %d (base %d)\n", p.EffectiveAtk, p.Attack))
	b.WriteString(fmt.Sprintf("  DEF    : %d (base %d)\n", p.EffectiveDef, p.Defense))
	b.WriteString(fmt.Sprintf("  Niveau : %d   XP : %d/%d\n", p.Level, p.XP, game.XPNeededForLevel(p.Level+1)))
	b.WriteString(fmt.Sprintf("  Kills  : %d   Tours : %d\n", p.Kills, p.Turns))
	b.WriteString(fmt.Sprintf("\n── Objets (%d) ──\n", len(p.Inventory)))
	if len(p.Inventory) == 0 {
		b.WriteString(styleDim.Render("  (inventaire vide — ramasse des objets au sol)\n"))
	}
	for i, it := range p.Inventory {
		cursor := "  "
		if i == m.invCursor {
			cursor = styleCursor.Render("> ")
		}
		b.WriteString(cursor + fmt.Sprintf("[%d] %s\n", i, itemDesc(it)))
	}
	b.WriteString("\n" + styleDim.Render("↑↓ naviguer · e équiper · u utiliser · d jeter · esc/i retour"))
	return b.String()
}

// itemDesc décrit un objet de l'inventaire (nom + effet).
func itemDesc(it *game.Item) string {
	switch it.Kind {
	case game.ItemWeapon:
		return fmt.Sprintf("%-20s (arme +%d ATK)", it.Name, it.Power)
	case game.ItemArmor:
		return fmt.Sprintf("%-20s (armure +%d DEF)", it.Name, it.Power)
	case game.ItemPotionHP:
		return fmt.Sprintf("%-20s (+%d PV)", it.Name, it.Power)
	case game.ItemPotionMana:
		return fmt.Sprintf("%-20s (+%d mana)", it.Name, it.Power)
	default:
		return it.Name
	}
}
