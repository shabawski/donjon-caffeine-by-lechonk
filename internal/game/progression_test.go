package game

import "testing"

func TestXPNeededForLevel(t *testing.T) {
	// Paliers 30*n : niv2=30, niv3=90, niv4=180, niv5=300, niv6=450
	cases := map[int]int{1: 0, 2: 30, 3: 90, 4: 180, 5: 300, 6: 450}
	for lvl, want := range cases {
		if got := XPNeededForLevel(lvl); got != want {
			t.Errorf("XPNeededForLevel(%d) = %d, attendu %d", lvl, got, want)
		}
	}
}

func TestGainXPLevelUp(t *testing.T) {
	s := newTestState(1, 10, 10)
	p := &s.Player
	p.GainXP(s, 30) // palier niveau 2 = 30 XP
	if p.Level != 2 {
		t.Fatalf("niveau attendu 2, got %d", p.Level)
	}
	if p.MaxHP != 50 || p.HP != 50 {
		t.Errorf("PV attendu 50/50, got %d/%d", p.HP, p.MaxHP)
	}
	if p.Attack != 6 {
		t.Errorf("attaque attendue 6, got %d", p.Attack)
	}
	if p.Defense != 2 {
		t.Errorf("défense attendue 2, got %d", p.Defense)
	}
	if p.MaxMana != 24 || p.Mana != 24 {
		t.Errorf("mana attendu 24/24, got %d/%d", p.Mana, p.MaxMana)
	}
}

func TestGainXPMultipleLevels(t *testing.T) {
	s := newTestState(1, 10, 10)
	p := &s.Player
	p.GainXP(s, 90) // paliers 30 (niv 2) + 60 (niv 3)
	if p.Level != 3 {
		t.Fatalf("niveau attendu 3, got %d", p.Level)
	}
}

func TestGainXPCapsAtLevel20(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.GainXP(s, 1_000_000)
	if s.Player.Level != 20 {
		t.Fatalf("le niveau doit être plafonné à 20, got %d", s.Player.Level)
	}
}

func TestGainXPNoLevelUpBelowThreshold(t *testing.T) {
	s := newTestState(1, 10, 10)
	p := &s.Player
	p.GainXP(s, 29) // sous le palier 30 (niveau 2)
	if p.Level != 1 {
		t.Fatalf("pas de montée de niveau sous le palier, got %d", p.Level)
	}
	if p.XP != 29 {
		t.Fatalf("XP cumulé attendu 29, got %d", p.XP)
	}
}

func TestEffectiveAttackDefense(t *testing.T) {
	s := newTestState(1, 10, 10)
	p := &s.Player
	if p.EffectiveAttack() != 5 || p.EffectiveDefense() != 1 {
		t.Fatalf("sans équipement: ATK %d DEF %d, attendu 5/1", p.EffectiveAttack(), p.EffectiveDefense())
	}
	p.WeaponBonus, p.ArmorBonus = 3, 2
	if p.EffectiveAttack() != 8 || p.EffectiveDefense() != 3 {
		t.Fatalf("avec équipement: ATK %d DEF %d, attendu 8/3", p.EffectiveAttack(), p.EffectiveDefense())
	}
}
