package game

// Merchant est un PNJ marchand présent sur certains étages.
// Il achète et vend des objets contre des pièces d'or.
type Merchant struct {
	X, Y  int
	Glyph rune
	Name  string
	Shop  []ShopItem // items en vente (générés à l'apparition)
}

// ShopItem est un objet en vente chez le marchand, avec son prix.
type ShopItem struct {
	Item  *Item
	Price int
}

// NewMerchant crée un marchand en (x, y) avec une boutique adaptée à la profondeur.
func NewMerchant(rng *RNG, x, y, depth int) *Merchant {
	m := &Merchant{X: x, Y: y, Glyph: 'M', Name: "marchand"}
	m.Shop = GenerateShop(rng, depth)
	return m
}

// GenerateShop génère la boutique d'un marchand : 2 armes, 2 armures, 2 potions
// dont les stats dépendent de la profondeur de l'étage.
func GenerateShop(rng *RNG, depth int) []ShopItem {
	shop := make([]ShopItem, 0, 6)
	// 2 armes
	for i := 0; i < 2; i++ {
		power := 1 + depth/2 + rng.Intn(2)
		names := []string{"dague", "épée courte", "hache", "marteau", "épée longue", "hache de guerre"}
		name := names[min(power-1, len(names)-1)]
		it := &Item{Kind: ItemWeapon, Name: name, Glyph: '/', Power: power}
		shop = append(shop, ShopItem{Item: it, Price: ItemValue(it)})
	}
	// 2 armures
	for i := 0; i < 2; i++ {
		power := 1 + depth/2 + rng.Intn(2)
		names := []string{"armure de cuir", "cotte de mailles", "plaque d'acier", "armure runique"}
		name := names[min(power-1, len(names)-1)]
		it := &Item{Kind: ItemArmor, Name: name, Glyph: '[', Power: power}
		shop = append(shop, ShopItem{Item: it, Price: ItemValue(it)})
	}
	// 2 potions
	for i := 0; i < 2; i++ {
		power := 20 + 5*depth
		if i == 1 {
			it := &Item{Kind: ItemPotionMana, Name: "potion de mana", Glyph: '!', Power: 15 + 3*depth}
			shop = append(shop, ShopItem{Item: it, Price: ItemValue(it)})
		} else {
			it := &Item{Kind: ItemPotionHP, Name: "potion de soin", Glyph: '!', Power: power}
			shop = append(shop, ShopItem{Item: it, Price: ItemValue(it)})
		}
	}
	return shop
}

// MerchantSellPrice renvoie le prix de revente d'un objet (50% de sa valeur).
func MerchantSellPrice(it *Item) int {
	return ItemValue(it) / 2
}

// BuyItem achète l'objet à l'index du shop du marchand de la map courante.
// L'objet est ajouté à l'inventaire du joueur ; l'or est débité.
// Retourne false si pas assez d'or, index invalide, ou pas de marchand.
func (s *State) BuyItem(shopIndex int) bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil || m.Merchant == nil {
		return false
	}
	merch := m.Merchant
	if shopIndex < 0 || shopIndex >= len(merch.Shop) {
		return false
	}
	si := merch.Shop[shopIndex]
	if s.Player.Gold < si.Price {
		s.Logf("Pas assez d'or (%d/%d).", s.Player.Gold, si.Price)
		return false
	}
	s.Player.Gold -= si.Price
	s.Player.Inventory = append(s.Player.Inventory, si.Item)
	s.Logf("💰 Tu achètes %s pour %d or.", si.Item.Name, si.Price)
	// Retire l'objet du shop (swap-remove).
	merch.Shop = append(merch.Shop[:shopIndex], merch.Shop[shopIndex+1:]...)
	return true
}

// SellItem vend l'objet à l'index de l'inventaire du joueur au marchand
// de la map courante. L'or est crédité (50% de la valeur).
// Retourne false si index invalide ou pas de marchand.
func (s *State) SellItem(invIndex int) bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil || m.Merchant == nil {
		return false
	}
	if invIndex < 0 || invIndex >= len(s.Player.Inventory) {
		return false
	}
	it := s.Player.Inventory[invIndex]
	price := MerchantSellPrice(it)
	s.Player.Gold += price
	s.Logf("💰 Tu vends %s pour %d or.", it.Name, price)
	s.Player.Inventory = append(s.Player.Inventory[:invIndex], s.Player.Inventory[invIndex+1:]...)
	return true
}

// MerchantAdjacent indique si le joueur est adjacent au marchand de sa map courante.
func (s *State) MerchantAdjacent() bool {
	m := s.curMap()
	if m == nil || m.Merchant == nil {
		return false
	}
	return chebDist(s.Player.X, s.Player.Y, m.Merchant.X, m.Merchant.Y) <= 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
