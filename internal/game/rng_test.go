package game

import "testing"

func TestRNGDeterministic(t *testing.T) {
	a, b := NewRNG(42), NewRNG(42)
	for i := 0; i < 1000; i++ {
		if a.Intn(1000) != b.Intn(1000) {
			t.Fatalf("séquences divergentes au tirage %d", i)
		}
	}
}

func TestRNGDifferentSeeds(t *testing.T) {
	a, b := NewRNG(1), NewRNG(2)
	same := 0
	for i := 0; i < 100; i++ {
		if a.Intn(1<<30) == b.Intn(1<<30) {
			same++
		}
	}
	if same > 5 {
		t.Fatalf("graines différentes, séquences trop similaires (%d/100)", same)
	}
}

func TestRNGIntRange(t *testing.T) {
	r := NewRNG(3)
	for i := 0; i < 10000; i++ {
		v := r.IntRange(5, 10)
		if v < 5 || v > 10 {
			t.Fatalf("hors bornes [5,10]: %d", v)
		}
	}
	if v := r.IntRange(10, 5); v < 5 || v > 10 {
		t.Fatalf("bornes échangées non gérées: %d", v)
	}
	if v := r.IntRange(7, 7); v != 7 {
		t.Fatalf("min == max doit donner %d, got %d", 7, v)
	}
}

func TestRNGChance(t *testing.T) {
	r := NewRNG(4)
	for i := 0; i < 100; i++ {
		if r.Chance(0) {
			t.Fatal("p=0 ne doit jamais réussir")
		}
	}
	for i := 0; i < 100; i++ {
		if !r.Chance(1) {
			t.Fatal("p=1 doit toujours réussir")
		}
	}
}

func TestRNGIntnNonPositive(t *testing.T) {
	r := NewRNG(5)
	if v := r.Intn(0); v != 0 {
		t.Fatalf("Intn(0) doit donner 0, got %d", v)
	}
	if v := r.Intn(-3); v != 0 {
		t.Fatalf("Intn(-3) doit donner 0, got %d", v)
	}
}
