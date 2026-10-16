package game

import "math/rand"

// RNG est une source de hasard déterministe et seedable.
// Deux parties lancées avec la même graine produisent exactement le même donjon.
type RNG struct {
	r *rand.Rand
}

// NewRNG crée un RNG avec la graine donnée.
func NewRNG(seed int64) *RNG {
	return &RNG{r: rand.New(rand.NewSource(seed))}
}

// Intn renvoie un entier dans [0, n). Renvoie 0 si n <= 0.
func (g *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return g.r.Intn(n)
}

// IntRange renvoie un entier dans [min, max] inclus (bornes échangées si besoin).
func (g *RNG) IntRange(min, max int) int {
	if max < min {
		min, max = max, min
	}
	return min + g.r.Intn(max-min+1)
}

// Float renvoie un flottant dans [0, 1).
func (g *RNG) Float() float64 { return g.r.Float64() }

// Chance renvoie true avec une probabilité p (0..1).
func (g *RNG) Chance(p float64) bool { return g.r.Float64() < p }

// Shuffle mélange les indices [0, n) via l'algorithme de Fisher-Yates.
func (g *RNG) Shuffle(n int, swap func(i, j int)) {
	g.r.Shuffle(n, swap)
}
