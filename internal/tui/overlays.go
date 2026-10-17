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
		{"i", "inventaire / équipement"},
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

// viewInventory rend l'équipement et les statistiques du joueur.
func (m *Model) viewInventory() string {
	p := &m.state.Player
	var b strings.Builder
	b.WriteString(styleTitle.Render("🎒 Inventaire") + "\n\n")
	b.WriteString(fmt.Sprintf("  Arme   : %s\n", equipName(p.WeaponName, p.WeaponBonus)))
	b.WriteString(fmt.Sprintf("  Armure : %s\n", equipName(p.ArmorName, p.ArmorBonus)))
	b.WriteString(fmt.Sprintf("  Or     : %d pièces\n", p.Gold))
	b.WriteString(fmt.Sprintf("  PV     : %d/%d\n", p.HP, p.MaxHP))
	b.WriteString(fmt.Sprintf("  Mana   : %d/%d\n", p.Mana, p.MaxMana))
	b.WriteString(fmt.Sprintf("  ATK    : %d (base %d)\n", p.EffectiveAttack(), p.Attack))
	b.WriteString(fmt.Sprintf("  DEF    : %d (base %d)\n", p.EffectiveDefense(), p.Defense))
	b.WriteString(fmt.Sprintf("  Niveau : %d   XP : %d/%d\n", p.Level, p.XP, game.XPNeededForLevel(p.Level+1)))
	b.WriteString(fmt.Sprintf("  Kills  : %d   Tours : %d\n", p.Kills, p.Turns))
	b.WriteString("\n" + styleDim.Render("esc/i/entrée : retour au jeu"))
	return b.String()
}
