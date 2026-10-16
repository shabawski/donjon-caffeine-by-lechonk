package game

// ItemKind identifie un objet ramassable.
type ItemKind int

const (
	ItemGold ItemKind = iota
	ItemPotionHP
	ItemPotionMana
	ItemWeapon
	ItemArmor
)

// Item est un objet posé sur le sol d'un étage.
type Item struct {
	Kind  ItemKind
	Name  string
	Glyph rune
	X, Y  int
	Power int // valeur (or), soin (potions HP), mana (potions mana), bonus (équipement)
}

// RollItem crée un objet adapté à la profondeur de l'étage.
func RollItem(rng *RNG, x, y, depth int) *Item {
	roll := rng.Intn(100)
	switch {
	case roll < 30:
		return &Item{Kind: ItemGold, Name: "pièces d'or", Glyph: '$', X: x, Y: y,
			Power: rng.IntRange(5, 15) * depth}
	case roll < 50:
		return &Item{Kind: ItemPotionHP, Name: "potion de soin", Glyph: '!', X: x, Y: y,
			Power: 20 + 5*depth}
	case roll < 62:
		return &Item{Kind: ItemPotionMana, Name: "potion de mana", Glyph: '!', X: x, Y: y,
			Power: 15 + 3*depth}
	case roll < 82:
		weapons := []struct {
			name  string
			power int
		}{
			{"dague rouillée", 1},
			{"épée courte", 2},
			{"hache de barbare", 3},
			{"marteau de forge", 4},
		}
		idx := rng.Intn(len(weapons))
		if depth > 4 && idx < len(weapons)-1 && rng.Chance(0.5) {
			idx++ // meilleure qualité en profondeur
		}
		wp := weapons[idx]
		return &Item{Kind: ItemWeapon, Name: wp.name, Glyph: '/', X: x, Y: y, Power: wp.power}
	default:
		armors := []struct {
			name  string
			power int
		}{
			{"armure de cuir", 1},
			{"cotte de mailles", 2},
			{"plaque d'acier", 3},
		}
		idx := rng.Intn(len(armors))
		if depth > 4 && idx < len(armors)-1 && rng.Chance(0.5) {
			idx++
		}
		ar := armors[idx]
		return &Item{Kind: ItemArmor, Name: ar.name, Glyph: '[', X: x, Y: y, Power: ar.power}
	}
}

// RollItemCount renvoie le nombre d'objets déposés sur un étage.
func RollItemCount(rng *RNG, depth int) int {
	return 3 + rng.Intn(3) + depth/2
}
