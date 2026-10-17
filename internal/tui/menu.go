package tui

import (
	"fmt"
	"strings"

	"game-test/internal/score"
)

// viewMenu rend l'écran d'accueil avec le pitch du jeu.
func (m *Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("   ☕ LE DONJON CAFÉINÉ ☕") + "\n\n")
	b.WriteString("   Le Seigneur des Tasses a volé tout le café de l'équipe.\n")
	b.WriteString("   Descends les 10 étages du donjon, bats le boss, récupère le café.\n")
	b.WriteString("\n\n")
	items := []string{"Nouvelle partie (solo)", "Multijoueur (TCP)", "Meilleurs scores", "Quitter"}
	for i, it := range items {
		if i == m.menuCursor {
			b.WriteString(styleCursor.Render("> "+it) + "\n")
		} else {
			b.WriteString("  " + it + "\n")
		}
	}
	b.WriteString("\n" + styleDim.Render("↑↓ naviguer · entrée valider · q quitter"))
	return b.String()
}

// viewScores rend le tableau des meilleurs scores persistés (top 10).
func (m *Model) viewScores() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("🏆 Meilleurs scores") + "\n\n")
	entries := score.Top(10)
	if len(entries) == 0 {
		b.WriteString(styleDim.Render("Aucun score enregistré. Joue une partie !") + "\n")
	} else {
		for i, e := range entries {
			result := styleGameOver.Render("DÉFAITE ")
			if e.Won {
				result = styleVictory.Render("VICTOIRE")
			}
			b.WriteString(fmt.Sprintf("%2d. %6d pts  %s  étage %d, niv %d, %d kills, %d tours  (%s, seed %d)\n",
				i+1, e.Score, result, e.Depth, e.Level, e.Kills, e.Turns, e.Date, e.Seed))
		}
	}
	b.WriteString("\n" + styleDim.Render("esc/q/entrée : retour au menu"))
	return b.String()
}
