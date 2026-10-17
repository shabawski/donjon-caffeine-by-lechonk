package tui

import (
	"fmt"
	"strings"

	"game-test/internal/game"
)

// viewGameOver rend l'écran de fin (victoire ou défaite) avec le récapitulatif.
func (m *Model) viewGameOver() string {
	s := m.snapshot
	var b strings.Builder
	if s.Phase == game.PhaseVictory {
		b.WriteString(styleVictory.Render("🏆 VICTOIRE !") + "\n\n")
		b.WriteString("Tu as vaincu le Seigneur des Tasses et récupéré le café ! ☕\n\n")
	} else {
		b.WriteString(styleGameOver.Render("💀 GAME OVER") + "\n\n")
		b.WriteString(fmt.Sprintf("Tu es mort : %s.\n\n", s.CauseOfDeath))
	}
	b.WriteString(fmt.Sprintf("  Étage %d/%d · Niveau %d · Or %d · Kills %d · Tours %d\n",
		s.Depth, game.MaxDepth, s.Player.Level, s.Player.Gold, s.Player.Kills, s.Player.Turns))
	b.WriteString(fmt.Sprintf("  Score : %d\n\n", s.Score()))
	b.WriteString(styleCursor.Render("[r]") + " Rejouer   " +
		styleCursor.Render("[s]") + " Scores   " +
		styleCursor.Render("[q]") + " Quitter")
	return b.String()
}
