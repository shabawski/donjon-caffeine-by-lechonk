package game

import "testing"

func TestAttackAlwaysAtLeastOneDamage(t *testing.T) {
	rng := NewRNG(1)
	def := &Entity{HP: 1000, MaxHP: 1000, Defense: 1000}
	atk := &Entity{Attack: 1}
	dmg, _, killed := Attack(rng, atk, def)
	if dmg < 1 {
		t.Fatalf("dégâts minimum 1, got %d", dmg)
	}
	if killed {
		t.Fatal("la cible ne doit pas mourir")
	}
	if def.HP != 999 {
		t.Fatalf("HP attendu 999, got %d", def.HP)
	}
}

func TestAttackDefenseReducesDamage(t *testing.T) {
	// Même graine => mêmes tirages de variance/crit pour les deux attaques.
	rngA, rngB := NewRNG(77), NewRNG(77)
	weak := &Entity{HP: 1000, MaxHP: 1000, Defense: 0}
	strong := &Entity{HP: 1000, MaxHP: 1000, Defense: 6}
	atk := &Entity{Attack: 10}
	dmgA, _, _ := Attack(rngA, atk, weak)
	dmgB, _, _ := Attack(rngB, atk, strong)
	if dmgA <= dmgB {
		t.Fatalf("la défense doit réduire les dégâts: %d (def 0) vs %d (def 6)", dmgA, dmgB)
	}
}

func TestAttackKillsAndClampsHP(t *testing.T) {
	rng := NewRNG(1)
	def := &Entity{HP: 3, MaxHP: 10, Defense: 0}
	atk := &Entity{Attack: 100}
	dmg, _, killed := Attack(rng, atk, def)
	if dmg < 3 {
		t.Fatalf("dégâts %d, attendu >= 3", dmg)
	}
	if !killed {
		t.Fatal("la cible avec 3 PV doit mourir")
	}
	if def.HP != 0 {
		t.Fatalf("HP doit être borné à 0, got %d", def.HP)
	}
}

func TestAttackCritOccurs(t *testing.T) {
	rng := NewRNG(5)
	crits := 0
	const n = 5000
	for i := 0; i < n; i++ {
		def := &Entity{HP: 1 << 30, MaxHP: 1 << 30, Defense: 0}
		atk := &Entity{Attack: 5}
		_, crit, _ := Attack(rng, atk, def)
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
		def := &Entity{HP: 1 << 30, MaxHP: 1 << 30, Defense: 0}
		atk := &Entity{Attack: 7}
		dmg, _, _ := Attack(rng, atk, def)
		if dmg < 1 {
			t.Fatalf("dégâts négatifs ou nuls: %d", dmg)
		}
	}
}
