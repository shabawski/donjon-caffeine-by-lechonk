package game

import "testing"

func TestAttackAlwaysAtLeastOneDamage(t *testing.T) {
	rng := NewRNG(1)
	hp := 999 // 1000 - 1 (dégâts min 1)
	atk := 1
	def := 1000
	dmg, _, killed := Attack(rng, atk, def, &hp)
	if dmg < 1 {
		t.Fatalf("dégâts minimum 1, got %d", dmg)
	}
	if killed {
		t.Fatal("la cible ne doit pas mourir")
	}
	if hp != 998 {
		t.Fatalf("HP attendu 998, got %d", hp)
	}
}

func TestAttackDefenseReducesDamage(t *testing.T) {
	// Même graine => mêmes tirages de variance/crit pour les deux attaques.
	rngA, rngB := NewRNG(77), NewRNG(77)
	hpWeak, hpStrong := 1000, 1000
	atk := 10
	dmgA, _, _ := Attack(rngA, atk, 0, &hpWeak)
	dmgB, _, _ := Attack(rngB, atk, 6, &hpStrong)
	if dmgA <= dmgB {
		t.Fatalf("la défense doit réduire les dégâts: %d (def 0) vs %d (def 6)", dmgA, dmgB)
	}
}

func TestAttackKillsAndClampsHP(t *testing.T) {
	rng := NewRNG(1)
	hp := 3
	dmg, _, killed := Attack(rng, 100, 0, &hp)
	if dmg < 3 {
		t.Fatalf("dégâts %d, attendu >= 3", dmg)
	}
	if !killed {
		t.Fatal("la cible avec 3 PV doit mourir")
	}
	if hp != 0 {
		t.Fatalf("HP doit être borné à 0, got %d", hp)
	}
}

func TestAttackCritOccurs(t *testing.T) {
	rng := NewRNG(5)
	crits := 0
	const n = 5000
	for i := 0; i < n; i++ {
		hp := 1 << 30
		_, crit, _ := Attack(rng, 5, 0, &hp)
		if crit {
			crits++
		}
	}
	// 10% attendu ; bornes larges [2%, 25%].
	if crits < n/50 || crits > n/4 {
		t.Fatalf("taux de crit anormal: %d/%d", crits, n)
	}
}

func TestAttackVarianceStaysPositive(t *testing.T) {
	rng := NewRNG(9)
	for i := 0; i < 1000; i++ {
		hp := 1 << 30
		dmg, _, _ := Attack(rng, 7, 0, &hp)
		if dmg < 1 {
			t.Fatalf("dégâts négatifs ou nuls: %d", dmg)
		}
	}
}
