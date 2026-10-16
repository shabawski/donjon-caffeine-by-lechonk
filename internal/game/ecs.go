package game

// EntityID identifie une entité dans le registre ECS.
type EntityID int

// InvalidEntity est l'identifiant nul (aucune entité).
const InvalidEntity EntityID = -1

// Entities est le registre ECS des monstres de l'étage courant.
//
// Architecture SOA (Structure of Arrays) : chaque composant est un tableau
// parallèle indexé par EntityID. Les itérations de l'IA et du FOV parcourent
// des tranches contiguës (xs, ys, hps…) — localité cache optimale, pas
// d'indirection pointeur, pas d'allocation par entité.
//
// Le registre compacte les entités mortes en fin de tour (Compact) pour
// libérer la RAM et garder les tableaux denses.
type Entities struct {
	Kind        []EntityKind
	Name        []string
	Glyph       []rune
	X, Y        []int
	MaxHP, HP   []int
	Attack      []int
	Defense     []int
	XPValue     []int
	DetectRange []int
	FleeBelow   []float64
	Alive       []bool
}

// NewEntities crée un registre vide.
func NewEntities() *Entities { return &Entities{} }

// Len renvoie le nombre d'entités vivantes.
func (e *Entities) Len() int {
	n := 0
	for _, a := range e.Alive {
		if a {
			n++
		}
	}
	return n
}

// Spawn crée une entité à partir d'un template et retourne son ID.
func (e *Entities) Spawn(t Entity) EntityID {
	e.Kind = append(e.Kind, t.Kind)
	e.Name = append(e.Name, t.Name)
	e.Glyph = append(e.Glyph, t.Glyph)
	e.X = append(e.X, t.X)
	e.Y = append(e.Y, t.Y)
	e.MaxHP = append(e.MaxHP, t.MaxHP)
	e.HP = append(e.HP, t.HP)
	e.Attack = append(e.Attack, t.Attack)
	e.Defense = append(e.Defense, t.Defense)
	e.XPValue = append(e.XPValue, t.XPValue)
	e.DetectRange = append(e.DetectRange, t.DetectRange)
	e.FleeBelow = append(e.FleeBelow, t.FleeBelow)
	e.Alive = append(e.Alive, true)
	return EntityID(len(e.Alive) - 1)
}

// Despawn marque une entité comme morte (elle sera compactée par Compact).
func (e *Entities) Despawn(id EntityID) {
	i := int(id)
	if i >= 0 && i < len(e.Alive) {
		e.Alive[i] = false
		e.HP[i] = 0
	}
}

// IsAlive indique si l'entité est vivante.
func (e *Entities) IsAlive(id EntityID) bool {
	i := int(id)
	return i >= 0 && i < len(e.Alive) && e.Alive[i]
}

// EachAlive itère sur les entités vivantes (ordre stable = ordre de spawn).
func (e *Entities) EachAlive(fn func(id EntityID)) {
	for i, alive := range e.Alive {
		if alive {
			fn(EntityID(i))
		}
	}
}

// EnemyAt retourne l'ID du monstre vivant en (x, y), ou InvalidEntity.
func (e *Entities) EnemyAt(x, y int) EntityID {
	for i, alive := range e.Alive {
		if alive && e.X[i] == x && e.Y[i] == y {
			return EntityID(i)
		}
	}
	return InvalidEntity
}

// BossID retourne l'ID du boss vivant, ou InvalidEntity.
func (e *Entities) BossID() EntityID {
	for i, alive := range e.Alive {
		if alive && e.Kind[i] == KindBoss {
			return EntityID(i)
		}
	}
	return InvalidEntity
}

// Snapshot retourne une copie de l'entité (pour rendu/lecture externe).
func (e *Entities) Snapshot(id EntityID) Entity {
	i := int(id)
	return Entity{
		Kind:        e.Kind[i],
		Name:        e.Name[i],
		Glyph:       e.Glyph[i],
		X:           e.X[i],
		Y:           e.Y[i],
		MaxHP:       e.MaxHP[i],
		HP:          e.HP[i],
		Attack:      e.Attack[i],
		Defense:     e.Defense[i],
		XPValue:     e.XPValue[i],
		DetectRange: e.DetectRange[i],
		FleeBelow:   e.FleeBelow[i],
	}
}

// AllSnapshots retourne les snapshots de toutes les entités vivantes.
func (e *Entities) AllSnapshots() []Entity {
	out := make([]Entity, 0, e.Len())
	e.EachAlive(func(id EntityID) {
		out = append(out, e.Snapshot(id))
	})
	return out
}

// Reset vide le registre (nouvel étage). Les tableaux sont tronqués à zéro
// longueur mais conservent leur capacité (pas de réallocation au étage suivant).
func (e *Entities) Reset() {
	e.Kind = e.Kind[:0]
	e.Name = e.Name[:0]
	e.Glyph = e.Glyph[:0]
	e.X = e.X[:0]
	e.Y = e.Y[:0]
	e.MaxHP = e.MaxHP[:0]
	e.HP = e.HP[:0]
	e.Attack = e.Attack[:0]
	e.Defense = e.Defense[:0]
	e.XPValue = e.XPValue[:0]
	e.DetectRange = e.DetectRange[:0]
	e.FleeBelow = e.FleeBelow[:0]
	e.Alive = e.Alive[:0]
}

// Compact supprime physiquement les entités mortes : recopie les vivantes au
// début des tableaux puis tronque. Libère la RAM des monstres tués et garde
// les tableaux denses (pas de trous). Les EntityID sont invalidés après
// Compact — à appeler uniquement en fin de tour, hors de toute boucle active.
func (e *Entities) Compact() {
	write := 0
	for read := 0; read < len(e.Alive); read++ {
		if !e.Alive[read] {
			continue
		}
		if write != read {
			e.Kind[write] = e.Kind[read]
			e.Name[write] = e.Name[read]
			e.Glyph[write] = e.Glyph[read]
			e.X[write] = e.X[read]
			e.Y[write] = e.Y[read]
			e.MaxHP[write] = e.MaxHP[read]
			e.HP[write] = e.HP[read]
			e.Attack[write] = e.Attack[read]
			e.Defense[write] = e.Defense[read]
			e.XPValue[write] = e.XPValue[read]
			e.DetectRange[write] = e.DetectRange[read]
			e.FleeBelow[write] = e.FleeBelow[read]
		}
		e.Alive[write] = true
		write++
	}
	e.Kind = e.Kind[:write]
	e.Name = e.Name[:write]
	e.Glyph = e.Glyph[:write]
	e.X = e.X[:write]
	e.Y = e.Y[:write]
	e.MaxHP = e.MaxHP[:write]
	e.HP = e.HP[:write]
	e.Attack = e.Attack[:write]
	e.Defense = e.Defense[:write]
	e.XPValue = e.XPValue[:write]
	e.DetectRange = e.DetectRange[:write]
	e.FleeBelow = e.FleeBelow[:write]
	e.Alive = e.Alive[:write]
}
