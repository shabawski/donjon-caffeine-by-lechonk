package game

import "testing"

// TestFocusPlayerTargetsClosest vérifie que le monstre cible le joueur le plus
// proche détecté (focus multi), pas forcément le joueur local.
func TestFocusPlayerTargetsClosest(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 5, 5
	s.Player.ID = 0
	// Un autre joueur plus proche du monstre.
	other := &Player{Entity: *NewPlayer(10, 5), Level: 1, MaxMana: StartMaxMana, Mana: StartMaxMana}
	other.ID = 1
	s.Others = []*Player{other}
	// Monstre en (8,5) : dist 2 de l'autre joueur, dist 3 du local.
	gob := s.addEnemy(KindGoblin, 8, 5)
	tx, ty, ok := s.focusPlayer(gob)
	if !ok {
		t.Fatal("le monstre doit détecter un joueur")
	}
	if tx != 10 || ty != 5 {
		t.Fatalf("focus attendu sur l'autre joueur (10,5), got (%d,%d)", tx, ty)
	}
}

// TestFocusPlayerSolo vérifie qu'en solo, le focus est le joueur local.
func TestFocusPlayerSolo(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 5, 5
	gob := s.addEnemy(KindGoblin, 8, 5)
	tx, ty, ok := s.focusPlayer(gob)
	if !ok {
		t.Fatal("le monstre doit détecter le joueur local")
	}
	if tx != 5 || ty != 5 {
		t.Fatalf("focus solo attendu sur (5,5), got (%d,%d)", tx, ty)
	}
}

// TestEnemyTurnAttacksOtherPlayer vérifie qu'un monstre adjacent attaque
// l'autre joueur (pas le local) quand c'est le focus.
func TestEnemyTurnAttacksOtherPlayer(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 2, 2 // joueur local loin
	s.Player.HP = s.Player.MaxHP
	other := &Player{Entity: *NewPlayer(8, 5), Level: 1, MaxMana: StartMaxMana, Mana: StartMaxMana}
	other.ID = 1
	other.HP = 40
	other.MaxHP = 40
	s.Others = []*Player{other}
	rat := s.addEnemy(KindRat, 7, 5) // adjacent à l'autre joueur (8,5)
	ComputeFOV(s.World, s.FOV, 2, 2, PlayerViewRange)
	s.EnemyTurn(rat)
	if other.HP >= 40 {
		t.Fatal("le rat doit attaquer l'autre joueur (focus), pas le local")
	}
	if s.Player.HP != s.Player.MaxHP {
		t.Fatal("le joueur local ne doit pas être attaqué (pas le focus)")
	}
}

// TestSnapshotForPlayer vérifie que le snapshot est personnalisé pour le joueur
// donné (position, FOV, PlayerID) et que les autres joueurs sont dans Others.
func TestSnapshotForPlayer(t *testing.T) {
	s := newTestState(1, 20, 20)
	s.Player.X, s.Player.Y = 5, 5
	s.Player.ID = 0
	other := &Player{Entity: *NewPlayer(10, 5), Level: 1, MaxMana: StartMaxMana, Mana: StartMaxMana}
	other.ID = 1
	s.Others = []*Player{other}
	// FOV centré sur l'autre joueur.
	fov := NewFOV(20 * 20)
	ComputeFOV(s.World, fov, 10, 5, PlayerViewRange)
	snap := s.SnapshotForPlayer(other, fov, 1)
	if snap.PlayerID != 1 {
		t.Fatalf("PlayerID attendu 1, got %d", snap.PlayerID)
	}
	if snap.Player.X != 10 || snap.Player.Y != 5 {
		t.Fatalf("joueur attendu (10,5), got (%d,%d)", snap.Player.X, snap.Player.Y)
	}
	// L'autre joueur (local, ID 0) doit être dans Others.
	if len(snap.Others) != 1 || snap.Others[0].X != 5 {
		t.Fatalf("le joueur local doit être dans Others, got %v", snap.Others)
	}
	// Le FOV du snapshot doit être celui de l'autre joueur (sa case visible).
	if !snap.Visible[5*20+10] {
		t.Fatal("le FOV du snapshot doit être celui de l'autre joueur")
	}
}

// TestSnapshotForPlayerSolo vérifie qu'en solo, Others est vide.
func TestSnapshotForPlayerSolo(t *testing.T) {
	s := newTestState(1, 20, 20)
	snap := s.SnapshotForPlayer(&s.Player, s.FOV, 0)
	if len(snap.Others) != 0 {
		t.Fatalf("pas d'autres joueurs en solo, got %d", len(snap.Others))
	}
	if snap.PlayerID != 0 {
		t.Fatalf("PlayerID solo attendu 0, got %d", snap.PlayerID)
	}
}

// TestAllPlayers vérifie que AllPlayers retourne local + autres.
func TestAllPlayers(t *testing.T) {
	s := newTestState(1, 10, 10)
	if got := len(s.AllPlayers()); got != 1 {
		t.Fatalf("solo: 1 joueur attendu, got %d", got)
	}
	other := &Player{Entity: *NewPlayer(5, 5)}
	other.ID = 1
	s.Others = []*Player{other}
	if got := len(s.AllPlayers()); got != 2 {
		t.Fatalf("multi: 2 joueurs attendus, got %d", got)
	}
}
