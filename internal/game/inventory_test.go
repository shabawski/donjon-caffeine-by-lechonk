package game

import "testing"

// TestEquipItemWeapon vérifie qu'équiper une arme depuis l'inventaire
// applique le bonus et retire l'objet de l'inventaire.
func TestEquipItemWeapon(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.Inventory = []*Item{{Kind: ItemWeapon, Name: "épée courte", Glyph: '/', Power: 3}}
	if !s.EquipItem(0) {
		t.Fatal("EquipItem doit réussir")
	}
	if s.Player.WeaponName != "épée courte" || s.Player.WeaponBonus != 3 {
		t.Fatalf("arme attendue équipée, got %q (+%d)", s.Player.WeaponName, s.Player.WeaponBonus)
	}
	if len(s.Player.Inventory) != 0 {
		t.Fatal("l'arme équipée doit être retirée de l'inventaire")
	}
}

// TestEquipItemArmorSwapsOld vérifie que l'ancien équipement retourne dans
// l'inventaire quand on équipe un nouvel objet.
func TestEquipItemArmorSwapsOld(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.ArmorName = "armure de cuir"
	s.Player.ArmorBonus = 1
	s.Player.Inventory = []*Item{{Kind: ItemArmor, Name: "plaque d'acier", Glyph: '[', Power: 3}}
	if !s.EquipItem(0) {
		t.Fatal("EquipItem doit réussir")
	}
	if s.Player.ArmorName != "plaque d'acier" || s.Player.ArmorBonus != 3 {
		t.Fatalf("nouvelle armure attendue, got %q (+%d)", s.Player.ArmorName, s.Player.ArmorBonus)
	}
	if len(s.Player.Inventory) != 1 || s.Player.Inventory[0].Name != "armure de cuir" {
		t.Fatalf("l'ancienne armure doit retourner dans l'inventaire, got %v", s.Player.Inventory)
	}
}

// TestEquipItemInvalidIndex vérifie qu'un index invalide est refusé.
func TestEquipItemInvalidIndex(t *testing.T) {
	s := newTestState(1, 10, 10)
	if s.EquipItem(0) {
		t.Fatal("EquipItem sur inventaire vide doit échouer")
	}
	s.Player.Inventory = []*Item{{Kind: ItemPotionHP, Name: "potion", Glyph: '!', Power: 10}}
	if s.EquipItem(0) {
		t.Fatal("équiper une potion doit échouer (pas équipable)")
	}
}

// TestUseItemPotionHeals vérifie qu'utiliser une potion depuis l'inventaire
// soigne le joueur et consomme l'objet.
func TestUseItemPotionHeals(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.HP = 10
	s.Player.Inventory = []*Item{{Kind: ItemPotionHP, Name: "potion de soin", Glyph: '!', Power: 15}}
	if !s.UseItem(0) {
		t.Fatal("UseItem doit réussir")
	}
	if s.Player.HP != 25 {
		t.Fatalf("PV attendu 25 après potion, got %d", s.Player.HP)
	}
	if len(s.Player.Inventory) != 0 {
		t.Fatal("la potion utilisée doit être retirée de l'inventaire")
	}
}

// TestUseItemPotionClampsToMaxHP vérifie le clamp à MaxHP.
func TestUseItemPotionClampsToMaxHP(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.HP = s.Player.MaxHP - 5
	s.Player.Inventory = []*Item{{Kind: ItemPotionHP, Name: "potion de soin", Glyph: '!', Power: 20}}
	s.UseItem(0)
	if s.Player.HP != s.Player.MaxHP {
		t.Fatalf("PV doit être plafonné à %d, got %d", s.Player.MaxHP, s.Player.HP)
	}
}

// TestUseItemPotionMana vérifie qu'une potion de mana restaure le mana.
func TestUseItemPotionMana(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.Mana = 0
	s.Player.Inventory = []*Item{{Kind: ItemPotionMana, Name: "potion de mana", Glyph: '!', Power: 10}}
	if !s.UseItem(0) {
		t.Fatal("UseItem doit réussir")
	}
	if s.Player.Mana != 10 {
		t.Fatalf("mana attendu 10, got %d", s.Player.Mana)
	}
}

// TestDropItem vérifie que jeter un objet le pose au sol sous le joueur.
func TestDropItem(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.X, s.Player.Y = 3, 3
	s.Player.Inventory = []*Item{{Kind: ItemWeapon, Name: "dague", Glyph: '/', Power: 1}}
	s.DropItem(0)
	if len(s.Player.Inventory) != 0 {
		t.Fatal("l'objet jeté doit être retiré de l'inventaire")
	}
	it := s.ItemAt(3, 3)
	if it == nil || it.Name != "dague" {
		t.Fatalf("l'objet doit être posé au sol en (3,3), got %v", it)
	}
}

// TestInventoryValue vérifie le calcul de la valeur totale de l'inventaire.
func TestInventoryValue(t *testing.T) {
	s := newTestState(1, 10, 10)
	if got := s.InventoryValue(); got != 0 {
		t.Fatalf("inventaire vide: valeur 0 attendue, got %d", got)
	}
	s.Player.Inventory = []*Item{
		{Kind: ItemWeapon, Name: "épée", Glyph: '/', Power: 2},      // 20 + 2*15 = 50
		{Kind: ItemPotionHP, Name: "potion", Glyph: '!', Power: 20}, // 10 + 20 = 30
	}
	if got := s.InventoryValue(); got != 80 {
		t.Fatalf("valeur attendue 80, got %d", got)
	}
}

// TestMerchantSpawnOnEvenFloors vérifie que le marchand apparaît sur les
// étages pairs (hors dernier étage).
func TestMerchantSpawnOnEvenFloors(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		s := NewState(seed)
		// Descend jusqu'à l'étage 2 (pair) : marchand attendu.
		s.Player.X, s.Player.Y = s.World.StairsX, s.World.StairsY
		if !s.Descend() {
			t.Fatalf("seed %d: descente étage 2 impossible", seed)
		}
		if s.World.Merchant == nil {
			t.Fatalf("seed %d: marchand attendu à l'étage 2 (pair)", seed)
		}
		if len(s.World.Merchant.Shop) == 0 {
			t.Fatalf("seed %d: la boutique du marchand ne doit pas être vide", seed)
		}
	}
}

// TestMerchantAbsentOnOddFloors vérifie l'absence de marchand sur étages impairs.
func TestMerchantAbsentOnOddFloors(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		s := NewState(seed) // étage 1 (impair)
		if s.World.Merchant != nil {
			t.Fatalf("seed %d: pas de marchand attendu à l'étage 1 (impair)", seed)
		}
	}
}

// TestBuyItem vérifie l'achat : or débité, objet dans l'inventaire, retiré du shop.
func TestBuyItem(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.X, s.Player.Y = 3, 3
	s.World.Merchant = NewMerchant(NewRNG(1), 4, 3, 2)
	s.Player.Gold = 1000
	shopLen := len(s.World.Merchant.Shop)
	first := s.World.Merchant.Shop[0]
	price := first.Price
	if !s.BuyItem(0) {
		t.Fatal("l'achat doit réussir avec assez d'or")
	}
	if s.Player.Gold != 1000-price {
		t.Fatalf("or attendu %d, got %d", 1000-price, s.Player.Gold)
	}
	if len(s.Player.Inventory) != 1 || s.Player.Inventory[0].Name != first.Item.Name {
		t.Fatalf("l'objet acheté doit être dans l'inventaire, got %v", s.Player.Inventory)
	}
	if len(s.World.Merchant.Shop) != shopLen-1 {
		t.Fatalf("l'objet acheté doit être retiré du shop, %d restant(s)", len(s.World.Merchant.Shop))
	}
}

// TestBuyItemNotEnoughGold vérifie qu'un achat échoue sans assez d'or.
func TestBuyItemNotEnoughGold(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.World.Merchant = NewMerchant(NewRNG(1), 4, 3, 2)
	s.Player.Gold = 0
	if s.BuyItem(0) {
		t.Fatal("l'achat doit échouer sans or")
	}
	if s.Player.Gold != 0 {
		t.Fatal("l'or ne doit pas être débité en cas d'échec")
	}
	if len(s.Player.Inventory) != 0 {
		t.Fatal("l'inventaire ne doit pas changer en cas d'échec")
	}
}

// TestBuyItemNoMerchant vérifie qu'un achat échoue sans marchand.
func TestBuyItemNoMerchant(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.Gold = 1000
	if s.BuyItem(0) {
		t.Fatal("l'achat doit échouer sans marchand sur l'étage")
	}
}

// TestSellItem vérifie la vente : or crédité (50%), objet retiré de l'inventaire.
func TestSellItem(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.World.Merchant = NewMerchant(NewRNG(1), 4, 3, 2)
	it := &Item{Kind: ItemWeapon, Name: "épée courte", Glyph: '/', Power: 2} // valeur 50
	s.Player.Inventory = []*Item{it}
	s.Player.Gold = 10
	wantPrice := MerchantSellPrice(it) // 25
	if !s.SellItem(0) {
		t.Fatal("la vente doit réussir")
	}
	if s.Player.Gold != 10+wantPrice {
		t.Fatalf("or attendu %d, got %d", 10+wantPrice, s.Player.Gold)
	}
	if len(s.Player.Inventory) != 0 {
		t.Fatal("l'objet vendu doit être retiré de l'inventaire")
	}
}

// TestSellItemNoMerchant vérifie qu'une vente échoue sans marchand.
func TestSellItemNoMerchant(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.Inventory = []*Item{{Kind: ItemWeapon, Name: "épée", Glyph: '/', Power: 1}}
	if s.SellItem(0) {
		t.Fatal("la vente doit échouer sans marchand")
	}
	if len(s.Player.Inventory) != 1 {
		t.Fatal("l'inventaire ne doit pas changer en cas d'échec")
	}
}

// TestMerchantAdjacent vérifie la détection d'adjacence au marchand.
func TestMerchantAdjacent(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.X, s.Player.Y = 3, 3
	s.World.Merchant = &Merchant{X: 4, Y: 3, Glyph: 'M', Name: "marchand"}
	if !s.MerchantAdjacent() {
		t.Fatal("le joueur adjacent au marchand doit être détecté")
	}
	s.Player.X, s.Player.Y = 8, 8
	if s.MerchantAdjacent() {
		t.Fatal("le joueur loin du marchand ne doit pas être détecté")
	}
	s.World.Merchant = nil
	if s.MerchantAdjacent() {
		t.Fatal("sans marchand, MerchantAdjacent doit être false")
	}
}

// TestGenerateShop vérifie la composition de la boutique (2 armes, 2 armures, 2 potions).
func TestGenerateShop(t *testing.T) {
	shop := GenerateShop(NewRNG(1), 4)
	if len(shop) != 6 {
		t.Fatalf("boutique attendue avec 6 objets, got %d", len(shop))
	}
	counts := map[ItemKind]int{}
	for _, si := range shop {
		counts[si.Item.Kind]++
		if si.Price != ItemValue(si.Item) {
			t.Fatalf("prix %d != valeur %d pour %s", si.Price, ItemValue(si.Item), si.Item.Name)
		}
	}
	if counts[ItemWeapon] != 2 || counts[ItemArmor] != 2 {
		t.Fatalf("attendu 2 armes + 2 armures, got %v", counts)
	}
	if counts[ItemPotionHP]+counts[ItemPotionMana] != 2 {
		t.Fatalf("attendu 2 potions, got %v", counts)
	}
}
