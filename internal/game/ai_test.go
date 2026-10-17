package game

import "testing"

func TestBFSNextStepFindsPath(t *testing.T) {
	s := newTestState(1, 20, 20)
	nx, ny, ok := BFSNextStep(s.World, nil, 2, 2, 10, 10)
	if !ok {
		t.Fatal("un chemin est attendu dans une salle ouverte")
	}
	if abs(nx-2)+abs(ny-2) != 1 {
		t.Fatalf("le premier pas doit être adjacent au départ, got (%d,%d) depuis (2,2)", nx, ny)
	}
}

func TestBFSNextStepNoPathThroughWalls(t *testing.T) {
	s := newTestState(1, 20, 20)
	// Mur vertical complet en x=10 : la carte est coupée en deux.
	for y := 0; y < 20; y++ {
		s.World.Set(10, y, TileWall)
	}
	if _, _, ok := BFSNextStep(s.World, nil, 2, 2, 15, 15); ok {
		t.Fatal("aucun chemin ne doit traverser un mur")
	}
}

func TestBFSNextStepSameCell(t *testing.T) {
	s := newTestState(1, 10, 10)
	nx, ny, ok := BFSNextStep(s.World, nil, 3, 3, 3, 3)
	if !ok || nx != 3 || ny != 3 {
		t.Fatalf("même case: attendu (3,3) ok, got (%d,%d) ok=%v", nx, ny, ok)
	}
}

func TestBFSNextStepRespectsBlocked(t *testing.T) {
	s := newTestState(1, 10, 10)
	// Bloque tout sauf la case de départ.
	blocked := func(x, y int) bool { return !(x == 2 && y == 2) }
	if _, _, ok := BFSNextStep(s.World, blocked, 2, 2, 5, 5); ok {
		t.Fatal("chemin attendu bloqué par le callback")
	}
}

func TestBFSNextStepShortestPath(t *testing.T) {
	s := newTestState(1, 20, 20)
	// Le pas doit rapprocher de la cible (distance de Manhattan décroissante).
	nx, ny, ok := BFSNextStep(s.World, nil, 2, 2, 8, 2)
	if !ok {
		t.Fatal("chemin attendu")
	}
	dBefore := abs(8-2) + abs(2-2)
	dAfter := abs(8-nx) + abs(2-ny)
	if dAfter != dBefore-1 {
		t.Fatalf("le pas doit être optimal: distance %d -> %d", dBefore, dAfter)
	}
}

func TestEnemyTurnAttacksAdjacentPlayer(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.X, s.Player.Y = 5, 5
	s.Player.HP = s.Player.MaxHP
	rat := s.addEnemy(KindRat, 6, 5)
	ComputeFOV(s.World, s.FOV, 5, 5, PlayerViewRange)
	s.EnemyTurn(rat)
	if s.Player.HP >= s.Player.MaxHP {
		t.Fatal("un monstre adjacent doit attaquer le joueur")
	}
}

func TestEnemyTurnChasesVisiblePlayer(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 10, 10
	gob := s.addEnemy(KindGoblin, 10, 14) // distance 4, dans la portée de détection
	ComputeFOV(s.World, s.FOV, 10, 10, PlayerViewRange)
	before := chebDist(s.Entities.X[int(gob)], s.Entities.Y[int(gob)], 10, 10)
	s.EnemyTurn(gob)
	after := chebDist(s.Entities.X[int(gob)], s.Entities.Y[int(gob)], 10, 10)
	if after >= before {
		t.Fatalf("le monstre doit se rapprocher du joueur visible: %d -> %d", before, after)
	}
}

func TestEnemyTurnFleesWhenWeak(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 10, 10
	rat := s.addEnemy(KindRat, 10, 13) // distance 3
	s.Entities.HP[int(rat)] = 1        // 1/8 < FleeBelow (0.4) => fuite
	ComputeFOV(s.World, s.FOV, 10, 10, PlayerViewRange)
	before := chebDist(s.Entities.X[int(rat)], s.Entities.Y[int(rat)], 10, 10)
	s.EnemyTurn(rat)
	after := chebDist(s.Entities.X[int(rat)], s.Entities.Y[int(rat)], 10, 10)
	if after <= before {
		t.Fatalf("le rat blessé doit fuir le joueur: %d -> %d", before, after)
	}
}

func TestEnemyTurnPatrolsWhenPlayerHidden(t *testing.T) {
	s := newTestState(1, 30, 30)
	s.Player.X, s.Player.Y = 2, 2
	gob := s.addEnemy(KindGoblin, 25, 25) // distance ~23, hors détection
	ComputeFOV(s.World, s.FOV, 2, 2, PlayerViewRange)
	for i := 0; i < 20; i++ {
		s.EnemyTurn(gob)
	}
	if s.Player.HP != s.Player.MaxHP {
		t.Fatal("pas d'attaque si le joueur n'est pas détecté")
	}
}

func TestEnemyTurnDeadEnemyDoesNothing(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.X, s.Player.Y = 5, 5
	rat := s.addEnemy(KindRat, 6, 5)
	s.Entities.Despawn(rat)
	ComputeFOV(s.World, s.FOV, 5, 5, PlayerViewRange)
	s.EnemyTurn(rat)
	if s.Player.HP != s.Player.MaxHP {
		t.Fatal("un monstre mort ne doit pas attaquer")
	}
}

func TestBossAlwaysDetectsPlayer(t *testing.T) {
	s := newTestState(1, 40, 40)
	s.Player.X, s.Player.Y = 3, 3
	boss := s.addEnemy(KindBoss, 13, 3) // distance 10 : hors FOV (8), dans DetectRange (12)
	ComputeFOV(s.World, s.FOV, 3, 3, PlayerViewRange)
	if s.FOV.Visible[3*40+13] {
		t.Fatal("le boss à distance 10 est hors du FOV (portée 8)")
	}
	before := chebDist(s.Entities.X[int(boss)], s.Entities.Y[int(boss)], 3, 3)
	for i := 0; i < 3; i++ {
		s.EnemyTurn(boss)
	}
	after := chebDist(s.Entities.X[int(boss)], s.Entities.Y[int(boss)], 3, 3)
	if after >= before {
		t.Fatalf("le boss doit chasser même hors champ de vision: %d -> %d", before, after)
	}
}

// TestNonBossDoesNotDetectOutsideFOV : à la même distance, un monstre
// classique ne détecte rien (la règle spéciale appartient au boss).
func TestNonBossDoesNotDetectOutsideFOV(t *testing.T) {
	s := newTestState(1, 40, 40)
	s.Player.X, s.Player.Y = 3, 3
	gob := s.addEnemy(KindGoblin, 13, 3) // distance 10 : hors FOV et hors DetectRange (7)
	ComputeFOV(s.World, s.FOV, 3, 3, PlayerViewRange)
	for i := 0; i < 10; i++ {
		s.EnemyTurn(gob)
	}
	if s.Player.HP != s.Player.MaxHP {
		t.Fatal("un gobelin hors détection ne doit jamais attaquer à distance")
	}
}

// TestDetectsPlayerWallBlocksLOS : un mur entre le monstre et le joueur bloque
// la détection, même si le joueur est dans la portée de détection.
func TestDetectsPlayerWallBlocksLOS(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 5, 10
	// Mur vertical complet en x=10 : la carte est coupée en deux.
	for y := 0; y < 20; y++ {
		s.World.Set(10, y, TileWall)
	}
	gob := s.addEnemy(KindGoblin, 12, 10) // distance 7 = DetectRange, mais mur en x=10
	if s.detectsPlayer(gob) {
		t.Fatal("un mur doit bloquer la détection du gobelin")
	}
	// Sans mur entre les deux, le gobelin détecte le joueur.
	s.World.Set(10, 10, TileFloor)
	if !s.detectsPlayer(gob) {
		t.Fatal("sans mur, le gobelin doit détecter le joueur à distance 7")
	}
}

// TestDetectsPlayerOutOfRange : hors de la portée de détection, pas de détection
// même avec une ligne de vue dégagée.
func TestDetectsPlayerOutOfRange(t *testing.T) {
	s := newTestState(1, 30, 30)
	s.Player.X, s.Player.Y = 5, 15
	gob := s.addEnemy(KindGoblin, 20, 15) // distance 15 > DetectRange (7), LOS dégagée
	if s.detectsPlayer(gob) {
		t.Fatal("hors de portée, le gobelin ne doit pas détecter le joueur")
	}
}

// TestBossDetectsThroughWall : le boss flaire le joueur à travers les murs
// (règle spéciale du boss, dans sa portée de détection).
func TestBossDetectsThroughWall(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 5, 10
	for y := 0; y < 20; y++ {
		s.World.Set(10, y, TileWall)
	}
	boss := s.addEnemy(KindBoss, 15, 10) // distance 10 <= DetectRange (12), mur entre les deux
	if !s.detectsPlayer(boss) {
		t.Fatal("le boss doit flairer le joueur à travers les murs")
	}
}
