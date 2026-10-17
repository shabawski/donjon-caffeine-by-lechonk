package game

import (
	"testing"
)

// TestMapRegistryRoot vérifie que Root retourne la map sans parente.
func TestMapRegistryRoot(t *testing.T) {
	s := NewState(42)
	root := s.Maps.Root()
	if root == nil {
		t.Fatal("une map racine doit exister")
	}
	if root.ParentMapID != NoParentMapID {
		t.Fatalf("la racine doit avoir ParentMapID=%d, got %d", NoParentMapID, root.ParentMapID)
	}
	if root.Depth != 1 {
		t.Fatalf("racine COOP attendue à l'étage 1, got %d", root.Depth)
	}
}

// TestNewStateMMORPGOverworld vérifie la génération de l'overworld.
func TestNewStateMMORPGOverworld(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	if s.Mode != ModeMMORPG {
		t.Fatalf("mode attendu MMORPG, got %v", s.Mode)
	}
	m := s.curMap()
	if m == nil || m.Kind != MapKindOverworld {
		t.Fatal("la map racine doit être l'overworld")
	}
	if m.Depth != 0 {
		t.Fatalf("overworld attendue à depth 0, got %d", m.Depth)
	}
	if m.World.HasStairs || m.World.HasStairsUp {
		t.Fatal("pas d'escaliers en overworld")
	}
	if len(m.Buildings) != 3 {
		t.Fatalf("3 bâtiments attendus, got %d", len(m.Buildings))
	}
	if !m.World.IsWalkable(s.Player.X, s.Player.Y) {
		t.Fatal("le joueur doit spawner sur une case franchissable")
	}
	// Au moins un bâtiment donjon.
	found := false
	for _, b := range m.Buildings {
		if b.Kind == BuildingDungeonEntrance {
			found = true
		}
	}
	if !found {
		t.Fatal("au moins un bâtiment donjon attendu")
	}
}

// TestOverworldBuildingNotWalkable vérifie que les bâtiments sont infranchissables.
func TestOverworldBuildingNotWalkable(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	m := s.curMap()
	for _, b := range m.Buildings {
		if m.World.At(b.X, b.Y) != TileBuilding {
			t.Fatalf("bâtiment %q: tuile attendue TileBuilding, got %v", b.Name, m.World.At(b.X, b.Y))
		}
		if m.World.IsWalkable(b.X, b.Y) {
			t.Fatalf("bâtiment %q: la case doit être infranchissable", b.Name)
		}
	}
}

// TestEnterBuildingCreatesDungeonInstance vérifie l'entrée dans un donjon :
// l'instance est créée à la demande, peuplée, et liée au bâtiment.
func TestEnterBuildingCreatesDungeonInstance(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	mapsBefore := s.Maps.Len()
	if !s.EnterFirstDungeon() {
		t.Fatal("l'entrée dans le donjon doit réussir")
	}
	m := s.curMap()
	if m.Kind != MapKindBuilding {
		t.Fatalf("map attendue MapKindBuilding, got %v", m.Kind)
	}
	if m.Depth != 1 {
		t.Fatalf("donjon attendu à l'étage 1, got %d", m.Depth)
	}
	if !m.World.IsWalkable(s.Player.X, s.Player.Y) {
		t.Fatal("le joueur doit être sur une case franchissable")
	}
	if m.Entities.Len() < 1 {
		t.Fatal("des monstres doivent spawner dans le donjon")
	}
	if s.Maps.Len() != mapsBefore+1 {
		t.Fatalf("une instance de donjon doit être créée: %d -> %d", mapsBefore, s.Maps.Len())
	}
	// Le bâtiment est lié à l'instance.
	ow := s.Maps.Root()
	var linked *Building
	for _, b := range ow.Buildings {
		if b.InstanceMapID == m.ID {
			linked = b
		}
	}
	if linked == nil {
		t.Fatal("le bâtiment doit être lié à l'instance créée")
	}
}

// TestEnterBuildingTwiceReusesInstance vérifie qu'entrer deux fois dans le
// même bâtiment réutilise l'instance (pas de doublon de map).
func TestEnterBuildingTwiceReusesInstance(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	if !s.EnterFirstDungeon() {
		t.Fatal("première entrée")
	}
	instID := s.curMap().ID
	mapsAfterFirst := s.Maps.Len()
	if !s.ExitBuilding() {
		t.Fatal("sortie du donjon")
	}
	if s.curMap().Kind != MapKindOverworld {
		t.Fatal("le joueur doit être de retour sur l'overworld")
	}
	// Re-entre : même instance, pas de doublon.
	if !s.EnterFirstDungeon() {
		t.Fatal("deuxième entrée")
	}
	if s.curMap().ID != instID {
		t.Fatalf("l'instance doit être réutilisée: %d -> %d", instID, s.curMap().ID)
	}
	if s.Maps.Len() != mapsAfterFirst {
		t.Fatalf("aucune nouvelle map ne doit être créée: %d -> %d", mapsAfterFirst, s.Maps.Len())
	}
}

// TestExitBuildingReturnsNearBuilding vérifie qu'on réapparaît à côté du
// bâtiment en sortant du donjon.
func TestExitBuildingReturnsNearBuilding(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	if !s.EnterFirstDungeon() {
		t.Fatal("entrée")
	}
	instID := s.curMap().ID
	if !s.ExitBuilding() {
		t.Fatal("sortie")
	}
	m := s.curMap()
	if m.Kind != MapKindOverworld {
		t.Fatal("retour overworld attendu")
	}
	// Le joueur est adjacent au bâtiment (Chebyshev <= 1).
	var bx, by int
	found := false
	for _, b := range m.Buildings {
		if b.InstanceMapID == instID {
			bx, by, found = b.X, b.Y, true
		}
	}
	if !found {
		t.Fatal("bâtiment lié introuvable")
	}
	if chebDist(s.Player.X, s.Player.Y, bx, by) > 1 {
		t.Fatalf("le joueur doit réapparaître à côté du bâtiment (%d,%d), got (%d,%d)",
			bx, by, s.Player.X, s.Player.Y)
	}
}

// TestEnterBuildingNoBuilding vérifie l'échec loin de tout bâtiment.
func TestEnterBuildingNoBuilding(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	// Téléporte loin de tout bâtiment.
	m := s.curMap()
	placed := false
	for y := 1; y < m.World.Height-1 && !placed; y++ {
		for x := 1; x < m.World.Width-1 && !placed; x++ {
			if m.World.IsWalkable(x, y) && m.BuildingAdjacent(x, y) == nil {
				s.Player.X, s.Player.Y = x, y
				placed = true
			}
		}
	}
	if !placed {
		t.Fatal("pas de case loin des bâtiments trouvée")
	}
	mapsBefore := s.Maps.Len()
	if s.EnterBuilding() {
		t.Fatal("entrer sans bâtiment adjacent doit échouer")
	}
	if s.Maps.Len() != mapsBefore {
		t.Fatal("aucune map ne doit être créée")
	}
}

// TestEnterBuildingNotOverworld vérifie qu'on ne peut pas entrer/sortir en COOP.
func TestEnterBuildingNotOverworld(t *testing.T) {
	s := NewState(42) // COOP : donjon
	if s.EnterBuilding() {
		t.Fatal("EnterBuilding doit échouer hors overworld")
	}
	if s.ExitBuilding() {
		t.Fatal("ExitBuilding doit échouer hors instance de donjon")
	}
}

// TestDescendDoesNotAffectParentMap — RÉGRESSION (17/10 17:07) : descendre ne
// doit pas modifier la map parente (les autres joueurs dessus ne sont pas
// affectés) et remonter retourne sur la MÊME instance de map.
func TestDescendDoesNotAffectParentMap(t *testing.T) {
	s := NewState(42)
	m1 := s.curMap()
	n1 := m1.Entities.Len()
	// Un autre joueur reste sur l'étage 1.
	other := &Player{Entity: *NewPlayer(5, 5), MapID: m1.ID}
	other.ID = 1
	s.Others = []*Player{other}
	// Le joueur local descend.
	s.Player.X, s.Player.Y = m1.World.StairsX, m1.World.StairsY
	if !s.Descend() {
		t.Fatal("descente impossible")
	}
	// La map étage 1 est la même instance, avec ses entités intactes.
	if s.Maps.Get(m1.ID) != m1 {
		t.Fatal("la map étage 1 doit être la même instance")
	}
	if m1.Entities.Len() != n1 {
		t.Fatalf("les entités de l'étage 1 ne doivent pas changer: %d -> %d", n1, m1.Entities.Len())
	}
	if other.MapID != m1.ID {
		t.Fatal("l'autre joueur ne doit pas être déplacé")
	}
	// Remonter : retour sur la même instance étage 1.
	s.Player.X, s.Player.Y = s.curMap().World.StairsUpX, s.curMap().World.StairsUpY
	if !s.Ascend() {
		t.Fatal("remontée impossible")
	}
	if s.curMap() != m1 {
		t.Fatal("remonter doit retourner sur la même instance de la map étage 1")
	}
}

// TestEnemiesFocusOnlySameMap vérifie qu'un monstre ne cible que les joueurs
// sur SA map : un joueur plus proche sur une autre map est ignoré.
func TestEnemiesFocusOnlySameMap(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 5, 5
	// Un autre joueur sur une AUTRE map, plus proche du monstre.
	m2 := s.Maps.Create(MapKindDungeon, 2, s.curMap().ID, 0, 0)
	m2.World = GenerateWorld(NewRNG(99), 2, 20, 20)
	other := &Player{Entity: *NewPlayer(6, 5), MapID: m2.ID}
	other.ID = 1
	s.Others = []*Player{other}
	// Monstre en (8,5) : dist 2 de other (autre map), dist 3 du local.
	gob := s.addEnemy(KindGoblin, 8, 5)
	ComputeFOV(s.curMap().World, s.curFOV(), 5, 5, PlayerViewRange)
	tx, ty, ok := s.focusPlayer(gob)
	if !ok {
		t.Fatal("le monstre doit détecter le joueur local (même map)")
	}
	if tx != 5 || ty != 5 {
		t.Fatalf("le focus doit être le joueur local sur la même map (5,5), got (%d,%d)", tx, ty)
	}
}

// TestSnapshotMode vérifie que le snapshot expose le mode de jeu.
func TestSnapshotMode(t *testing.T) {
	coop := NewState(42)
	if snap := coop.Snapshot(); snap.Mode != ModeCoop {
		t.Fatalf("snapshot COOP: mode attendu ModeCoop, got %v", snap.Mode)
	}
	mmo := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	if snap := mmo.Snapshot(); snap.Mode != ModeMMORPG {
		t.Fatalf("snapshot MMORPG: mode attendu ModeMMORPG, got %v", snap.Mode)
	}
}

// TestMMORPGVictoryPathEndToEnd prouve la victoire en MMORPG de bout en bout :
// entrée dans le donjon, descente jusqu'au boss, kill.
func TestMMORPGVictoryPathEndToEnd(t *testing.T) {
	s := NewStateMMORPG(42, DefaultWidth, DefaultHeight)
	if !s.EnterFirstDungeon() {
		t.Fatal("entrée dans le donjon impossible")
	}
	for s.curMap().World.Depth < MaxDepth {
		s.Player.X, s.Player.Y = s.curMap().World.StairsX, s.curMap().World.StairsY
		if !s.Descend() {
			t.Fatalf("impossible de descendre à l'étage %d", s.curMap().World.Depth)
		}
	}
	boss := s.boss()
	if boss == InvalidEntity {
		t.Fatal("le boss doit exister au dernier étage")
	}
	s.Player.Attack = 1000
	s.Player.X, s.Player.Y = s.curMap().Entities.X[int(boss)], s.curMap().Entities.Y[int(boss)]
	s.MovePlayer(0, 0)
	if s.Phase != PhaseVictory {
		t.Fatalf("phase attendue PhaseVictory, got %v", s.Phase)
	}
	if !s.Won {
		t.Fatal("Won doit être true")
	}
}

// TestAutoPlayMMORPGTerminates vérifie qu'une partie MMORPG bot tourne de bout
// en bout sans panique (entrée dans le donjon + jeu automatique).
func TestAutoPlayMMORPGTerminates(t *testing.T) {
	stats := map[Phase]int{}
	for seed := int64(1); seed <= 5; seed++ {
		s := NewStateMMORPG(seed, DefaultWidth, DefaultHeight)
		if !s.EnterFirstDungeon() {
			t.Fatalf("seed %d: entrée dans le donjon impossible", seed)
		}
		for s.Phase == PhasePlaying && s.Player.Turns < 20000 {
			s.AutoAction()
		}
		stats[s.Phase]++
		_ = s.Summary()
		_ = s.Score()
	}
	t.Logf("5 seeds MMORPG: victoires=%d, gameover=%d, limite=%d",
		stats[PhaseVictory], stats[PhaseGameOver], stats[PhasePlaying])
	if stats[PhaseVictory]+stats[PhaseGameOver] == 0 {
		t.Error("aucune partie MMORPG terminée sur 5 seeds")
	}
}
