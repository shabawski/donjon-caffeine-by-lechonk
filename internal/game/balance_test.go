package game

import "testing"

// TestEnemyTemplates vérifie les stats des nouveaux monstres et du boss renforcé.
func TestEnemyTemplates(t *testing.T) {
	cases := []struct {
		kind            EntityKind
		name            string
		glyph           rune
		maxHP, atk, def int
		xp              int
		detectRange     int
	}{
		{KindRat, "rat géant", 'r', 8, 3, 0, 5, 6},
		{KindSlime, "slime", 'j', 6, 2, 0, 4, 5},
		{KindGoblin, "gobelin", 'g', 14, 5, 1, 12, 7},
		{KindBat, "chauve-souris", 'v', 10, 4, 0, 10, 8},
		{KindSkeleton, "squelette", 's', 22, 6, 3, 20, 6},
		{KindGolem, "golem de pierre", 'G', 40, 7, 6, 35, 5},
		{KindBoss, "Seigneur des Tasses", 'B', 120, 11, 5, 200, 12},
	}
	for _, tt := range cases {
		e := enemyTemplate(tt.kind)
		if e.Name != tt.name || e.Glyph != tt.glyph {
			t.Errorf("%v: nom/glyph = %q/%c, attendu %q/%c", tt.kind, e.Name, e.Glyph, tt.name, tt.glyph)
		}
		if e.MaxHP != tt.maxHP || e.HP != tt.maxHP || e.Attack != tt.atk || e.Defense != tt.def {
			t.Errorf("%v: stats = %d/%d PV, %d ATK, %d DEF, attendu %d PV, %d ATK, %d DEF",
				tt.kind, e.HP, e.MaxHP, e.Attack, e.Defense, tt.maxHP, tt.atk, tt.def)
		}
		if e.XPValue != tt.xp || e.DetectRange != tt.detectRange {
			t.Errorf("%v: XP=%d Detect=%d, attendu XP=%d Detect=%d", tt.kind, e.XPValue, e.DetectRange, tt.xp, tt.detectRange)
		}
	}
}

// TestBossIsStrongerThanV1 vérifie que le boss v2 a été renforcé (120 PV vs 60 en v1).
func TestBossIsStrongerThanV1(t *testing.T) {
	b := enemyTemplate(KindBoss)
	if b.MaxHP < 100 {
		t.Fatalf("boss trop faible : %d PV (attendu >= 100 après renforcement)", b.MaxHP)
	}
	if b.Attack < 10 {
		t.Fatalf("boss ATK trop faible : %d (attendu >= 10)", b.Attack)
	}
}

// TestEnemyKindsForDepth vérifie la répartition des monstres par profondeur.
func TestEnemyKindsForDepth(t *testing.T) {
	// Étages 1-2 : rat, slime, gobelin
	low := EnemyKindsForDepth(1)
	if !containsKind(low, KindRat) || !containsKind(low, KindSlime) || !containsKind(low, KindGoblin) {
		t.Fatalf("étage 1: attendu rat+slime+gobelin, got %v", low)
	}
	if containsKind(low, KindGolem) {
		t.Fatal("pas de golem aux étages 1-2")
	}
	// Étages 3-5 : chauve-souris ajoutée
	mid := EnemyKindsForDepth(4)
	if !containsKind(mid, KindBat) {
		t.Fatalf("étage 4: chauve-souris attendue, got %v", mid)
	}
	// Étages 8-10 : golem présent, pas de rat
	high := EnemyKindsForDepth(9)
	if !containsKind(high, KindGolem) {
		t.Fatalf("étage 9: golem attendu, got %v", high)
	}
	if containsKind(high, KindRat) {
		t.Fatal("pas de rat aux étages 8-10")
	}
}

func containsKind(kinds []EntityKind, k EntityKind) bool {
	for _, kk := range kinds {
		if kk == k {
			return true
		}
	}
	return false
}

// TestRollXPBounds vérifie que rollXP applique une variance bornée à ±20%.
func TestRollXPBounds(t *testing.T) {
	s := newTestState(1, 10, 10)
	const base = 10
	for i := 0; i < 1000; i++ {
		xp := s.rollXP(base)
		// ±20% de 10 → [8, 12]
		if xp < 8 || xp > 12 {
			t.Fatalf("rollXP(10) = %d hors bornes [8,12]", xp)
		}
	}
	// Base 0 ou négative → 0
	if xp := s.rollXP(0); xp != 0 {
		t.Fatalf("rollXP(0) = %d, attendu 0", xp)
	}
	// Base 1 → minimum 1 (jamais 0)
	for i := 0; i < 100; i++ {
		if xp := s.rollXP(1); xp < 1 {
			t.Fatalf("rollXP(1) = %d, minimum attendu 1", xp)
		}
	}
}

// TestRollXPDeterministic vérifie que rollXP est déterministe (même seed = même séquence).
func TestRollXPDeterministic(t *testing.T) {
	a, b := newTestState(42, 10, 10), newTestState(42, 10, 10)
	for i := 0; i < 100; i++ {
		if xa, xb := a.rollXP(10), b.rollXP(10); xa != xb {
			t.Fatalf("rollXP diverge au tirage %d: %d vs %d", i, xa, xb)
		}
	}
}

// TestAscendRequiresStairsUp vérifie que Ascend échoue sans escalier montant.
func TestAscendRequiresStairsUp(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.curMap().World.HasStairsUp = false
	if s.Ascend() {
		t.Fatal("remonter sans escalier montant doit échouer")
	}
	s.curMap().World.HasStairsUp = true
	s.curMap().World.StairsUpX, s.curMap().World.StairsUpY = 5, 5
	s.curMap().World.Set(5, 5, TileStairsUp)
	if s.Ascend() {
		t.Fatal("remonter hors de l'escalier montant doit échouer")
	}
	s.Player.X, s.Player.Y = 5, 5
	// newTestState est à l'étage 1 : pas d'escalier montant possible, on force l'étage 2
	s.curMap().Depth = 2
	if !s.Ascend() {
		t.Fatal("remonter sur l'escalier montant doit réussir")
	}
	if s.curMap().World.Depth != 1 {
		t.Fatalf("étage attendu 1 après Ascend, got %d", s.curMap().World.Depth)
	}
}

// TestAscendAtFirstFloorFails vérifie qu'on ne peut pas remonter depuis l'étage 1.
func TestAscendAtFirstFloorFails(t *testing.T) {
	s := NewState(42)
	// Étage 1 : pas d'escalier montant généré
	if s.curMap().World.HasStairsUp {
		t.Fatal("l'étage 1 ne doit pas avoir d'escalier montant")
	}
	if s.Ascend() {
		t.Fatal("Ascend doit échouer à l'étage 1")
	}
}

// TestGenerateWorldHasStairsUp vérifie la génération de l'escalier montant.
func TestGenerateWorldHasStairsUp(t *testing.T) {
	for seed := int64(0); seed < 10; seed++ {
		// Étage 1 : pas d'escalier montant
		w1 := GenerateWorld(NewRNG(seed), 1, 40, 20)
		if w1.HasStairsUp {
			t.Fatalf("seed %d: escalier montant présent à l'étage 1", seed)
		}
		// Étage 2+ : escalier montant présent et franchissable
		w2 := GenerateWorld(NewRNG(seed), 2, 40, 20)
		if !w2.HasStairsUp {
			t.Fatalf("seed %d: escalier montant absent à l'étage 2", seed)
		}
		if w2.At(w2.StairsUpX, w2.StairsUpY) != TileStairsUp {
			t.Fatalf("seed %d: case escalier montant = %v", seed, w2.At(w2.StairsUpX, w2.StairsUpY))
		}
		if !w2.IsWalkable(w2.StairsUpX, w2.StairsUpY) {
			t.Fatalf("seed %d: l'escalier montant doit être franchissable", seed)
		}
	}
}

// TestFloorStableAcrossAscendDescend vérifie que les maps persistent dans le
// registre : descendre à l'étage 2, remonter, redescendre → même instance de map
// (mêmes tuiles), sans régénération.
func TestFloorStableAcrossAscendDescend(t *testing.T) {
	s := NewState(42)
	// Descend à l'étage 2
	s.Player.X, s.Player.Y = s.curMap().World.StairsX, s.curMap().World.StairsY
	if !s.Descend() {
		t.Fatal("descente étage 2 impossible")
	}
	// Capture les tuiles de l'étage 2
	tilesFloor2 := make([]Tile, len(s.curMap().World.Tiles))
	copy(tilesFloor2, s.curMap().World.Tiles)
	stairs2 := [2]int{s.curMap().World.StairsX, s.curMap().World.StairsY}
	stairsUp2 := [2]int{s.curMap().World.StairsUpX, s.curMap().World.StairsUpY}
	// Remonte à l'étage 1 (map parente, persistée)
	s.Player.X, s.Player.Y = s.curMap().World.StairsUpX, s.curMap().World.StairsUpY
	if !s.Ascend() {
		t.Fatal("remontée étage 1 impossible")
	}
	if s.curMap().World.Depth != 1 {
		t.Fatalf("étage attendu 1, got %d", s.curMap().World.Depth)
	}
	// Redescend à l'étage 2 (même instance de map)
	s.Player.X, s.Player.Y = s.curMap().World.StairsX, s.curMap().World.StairsY
	if !s.Descend() {
		t.Fatal("redescente étage 2 impossible")
	}
	if s.curMap().World.Depth != 2 {
		t.Fatalf("étage attendu 2, got %d", s.curMap().World.Depth)
	}
	// Même carte ?
	for i := range tilesFloor2 {
		if s.curMap().World.Tiles[i] != tilesFloor2[i] {
			t.Fatalf("tuile %d diverge après remontée/redescente: %v vs %v", i, s.curMap().World.Tiles[i], tilesFloor2[i])
		}
	}
	if s.curMap().World.StairsX != stairs2[0] || s.curMap().World.StairsY != stairs2[1] {
		t.Fatal("position escalier étage 2 diverge")
	}
	if s.curMap().World.StairsUpX != stairsUp2[0] || s.curMap().World.StairsUpY != stairsUp2[1] {
		t.Fatal("position escalier montant étage 2 diverge")
	}
}

// TestTileStairsUpWalkable vérifie que l'escalier montant est franchissable.
func TestTileStairsUpWalkable(t *testing.T) {
	w := &World{Width: 5, Height: 5, Tiles: make([]Tile, 25)}
	w.Set(2, 2, TileStairsUp)
	if !w.IsWalkable(2, 2) {
		t.Fatal("TileStairsUp doit être franchissable")
	}
}
