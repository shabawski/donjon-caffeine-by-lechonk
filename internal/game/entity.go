package game

// EntityKind identifie le type d'une entité.
type EntityKind int

const (
	KindPlayer EntityKind = iota
	KindRat
	KindGoblin
	KindSkeleton
	KindBoss
)

// Entity est un personnage (joueur ou monstre) présent sur la carte.
type Entity struct {
	Kind        EntityKind
	Name        string
	Glyph       rune
	X, Y        int
	MaxHP, HP   int
	Attack      int
	Defense     int
	XPValue     int     // XP donné à sa mort (monstres uniquement)
	DetectRange int     // portée de détection du joueur (distance de Chebyshev)
	FleeBelow   float64 // fraction de PV sous laquelle l'ennemi fuit (0 = ne fuit jamais)
}

// IsAlive indique si l'entité a encore des PV.
func (e *Entity) IsAlive() bool { return e.HP > 0 }

// IsDead indique si l'entité est vaincue.
func (e *Entity) IsDead() bool { return e.HP <= 0 }

// NewPlayer crée l'entité du joueur (positionnée plus tard sur la carte).
func NewPlayer(x, y int) *Entity {
	return &Entity{
		Kind:    KindPlayer,
		Name:    "héros",
		Glyph:   '@',
		X:       x,
		Y:       y,
		MaxHP:   30,
		HP:      30,
		Attack:  5,
		Defense: 1,
	}
}

// enemyTemplate renvoie les statistiques de base d'un type de monstre.
func enemyTemplate(kind EntityKind) Entity {
	switch kind {
	case KindRat:
		return Entity{
			Kind: KindRat, Name: "rat géant", Glyph: 'r',
			MaxHP: 8, HP: 8, Attack: 3, Defense: 0,
			XPValue: 5, DetectRange: 6, FleeBelow: 0.4,
		}
	case KindGoblin:
		return Entity{
			Kind: KindGoblin, Name: "gobelin", Glyph: 'g',
			MaxHP: 14, HP: 14, Attack: 5, Defense: 1,
			XPValue: 12, DetectRange: 7,
		}
	case KindSkeleton:
		return Entity{
			Kind: KindSkeleton, Name: "squelette", Glyph: 's',
			MaxHP: 22, HP: 22, Attack: 6, Defense: 3,
			XPValue: 20, DetectRange: 6,
		}
	case KindBoss:
		return Entity{
			Kind: KindBoss, Name: "Seigneur des Tasses", Glyph: 'B',
			MaxHP: 60, HP: 60, Attack: 9, Defense: 4,
			XPValue: 200, DetectRange: 12,
		}
	default:
		return Entity{}
	}
}

// EnemyKindsForDepth renvoie les types de monstres pouvant apparaître à un étage.
func EnemyKindsForDepth(depth int) []EntityKind {
	switch {
	case depth <= 2:
		return []EntityKind{KindRat, KindGoblin}
	case depth <= 5:
		return []EntityKind{KindRat, KindGoblin, KindSkeleton}
	default:
		return []EntityKind{KindRat, KindGoblin, KindSkeleton}
	}
}
