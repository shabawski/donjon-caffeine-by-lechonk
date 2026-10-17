package tui

import (
	"fmt"
	"strings"

	"game-test/internal/game"
)

// viewShop rend l'écran du marchand : onglet achat (objets en vente avec prix)
// et onglet vente (inventaire du joueur avec prix de revente à 50%).
// S'ouvre avec la touche m quand le joueur est adjacent au PNJ marchand (M).
func (m *Model) viewShop() string {
	s := m.state
	merchant := s.World.Merchant
	var b strings.Builder
	b.WriteString(styleTitle.Render("💰 Marchand") + "\n\n")
	b.WriteString(fmt.Sprintf("  Or : %d pièces\n\n", s.Player.Gold))
	// Onglets achat / vente.
	tabAchat, tabVente := " Achat ", " Vente "
	if m.shopTab == 0 {
		tabAchat = styleCursor.Render("[Achat]")
		tabVente = styleDim.Render(tabVente)
	} else {
		tabAchat = styleDim.Render(tabAchat)
		tabVente = styleCursor.Render("[Vente]")
	}
	b.WriteString("  " + tabAchat + "  " + tabVente + "\n\n")
	if m.shopTab == 0 {
		// Onglet achat : liste de la boutique du marchand.
		if merchant == nil || len(merchant.Shop) == 0 {
			b.WriteString(styleDim.Render("  (plus rien à vendre)\n"))
		}
		for i, si := range merchant.Shop {
			cursor := "  "
			if i == m.shopCursor {
				cursor = styleCursor.Render("> ")
			}
			b.WriteString(cursor + fmt.Sprintf("[%d] %s %d or\n", i, itemDesc(si.Item), si.Price))
		}
	} else {
		// Onglet vente : liste de l'inventaire avec prix de revente.
		if len(s.Player.Inventory) == 0 {
			b.WriteString(styleDim.Render("  (inventaire vide)\n"))
		}
		for i, it := range s.Player.Inventory {
			cursor := "  "
			if i == m.shopCursor {
				cursor = styleCursor.Render("> ")
			}
			b.WriteString(cursor + fmt.Sprintf("[%d] %s %d or\n", i, itemDesc(it), game.MerchantSellPrice(it)))
		}
	}
	b.WriteString("\n" + styleDim.Render("↑↓ naviguer · tab achat/vente · entrée acheter/vendre · esc retour"))
	return b.String()
}
