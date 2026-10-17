package tui

import (
	"game-test/internal/game"

	"github.com/charmbracelet/lipgloss"
)

// Styles du jeu, définis une fois au niveau package.
// Sans TTY (tests), lipgloss rend du texte brut : les assertions restent possibles.
var (
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	stylePlayer      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	styleWall        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleFloor       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleStairs      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleStairsUp    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleMerchant    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	styleOtherPlayer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	styleDim         = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	styleHP          = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleMana        = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleHUD         = lipgloss.NewStyle().Padding(0, 1)
	styleLog         = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleVictory     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))
	styleGameOver    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleCursor      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)

var enemyStyles = map[game.EntityKind]lipgloss.Style{
	game.KindRat:      lipgloss.NewStyle().Foreground(lipgloss.Color("174")),
	game.KindGoblin:   lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
	game.KindSkeleton: lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
	game.KindSlime:    lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
	game.KindBat:      lipgloss.NewStyle().Foreground(lipgloss.Color("92")),
	game.KindGolem:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244")),
	game.KindBoss:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("201")),
}

var itemStyles = map[game.ItemKind]lipgloss.Style{
	game.ItemGold:       lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
	game.ItemPotionHP:   lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	game.ItemPotionMana: lipgloss.NewStyle().Foreground(lipgloss.Color("27")),
	game.ItemWeapon:     lipgloss.NewStyle().Foreground(lipgloss.Color("135")),
	game.ItemArmor:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
}

func styleForKind(k game.EntityKind) lipgloss.Style {
	if st, ok := enemyStyles[k]; ok {
		return st
	}
	return stylePlayer
}

func styleForItem(k game.ItemKind) lipgloss.Style {
	if st, ok := itemStyles[k]; ok {
		return st
	}
	return styleFloor
}
