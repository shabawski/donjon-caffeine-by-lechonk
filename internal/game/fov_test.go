package game

import "testing"

func TestFOVPlayerCellVisible(t *testing.T) {
	s := newTestState(1, 20, 20)
	if !s.FOV.Visible[1*20+1] {
		t.Fatal("le joueur doit voir sa propre case")
	}
	if !s.FOV.Explored[1*20+1] {
		t.Fatal("la case du joueur doit être explorée")
	}
}

func TestFOVWallsBlockVision(t *testing.T) {
	s := newTestState(1, 14, 14)
	// Mur vertical complet en x=6 : sépare la carte en deux.
	for y := 1; y < 13; y++ {
		s.World.Set(6, y, TileWall)
	}
	s.Player.X, s.Player.Y = 3, 7
	ComputeFOV(s.World, s.FOV, 3, 7, PlayerViewRange)
	if !s.FOV.Visible[7*14+5] {
		t.Fatal("la case (5,7) avant le mur doit être visible")
	}
	if s.FOV.Visible[7*14+9] {
		t.Fatal("la case (9,7) derrière le mur ne doit pas être visible")
	}
	if s.FOV.Visible[10*14+9] {
		t.Fatal("la case (9,10) derrière le mur ne doit pas être visible")
	}
}

func TestFOVExploredPersists(t *testing.T) {
	s := newTestState(1, 20, 20)
	if !s.FOV.Explored[1*20+1] {
		t.Fatal("case de départ explorée")
	}
	s.Player.X, s.Player.Y = 15, 15
	ComputeFOV(s.World, s.FOV, 15, 15, PlayerViewRange)
	if s.FOV.Visible[1*20+1] {
		t.Fatal("la case lointaine ne doit plus être visible")
	}
	if !s.FOV.Explored[1*20+1] {
		t.Fatal("une case explorée doit le rester (fog of war)")
	}
}

func TestFOVRespectsRange(t *testing.T) {
	s := newTestState(1, 30, 30)
	s.Player.X, s.Player.Y = 15, 15
	ComputeFOV(s.World, s.FOV, 15, 15, PlayerViewRange)
	// (23,15) : distance 8, dans la portée, ligne droite dégagée.
	if !s.FOV.Visible[15*30+23] {
		t.Fatal("une case à distance 8 en ligne droite doit être visible")
	}
	// (24,15) : distance 9, hors de la portée.
	if s.FOV.Visible[15*30+24] {
		t.Fatal("une case à distance 9 est hors de portée")
	}
}

func TestFOVUnexploredIsNotVisible(t *testing.T) {
	s := newTestState(1, 30, 30)
	s.Player.X, s.Player.Y = 15, 15
	ComputeFOV(s.World, s.FOV, 15, 15, PlayerViewRange)
	// (28,2) : hors de portée du spawn (1,1) et de (15,15), jamais vu.
	if s.FOV.Visible[2*30+28] || s.FOV.Explored[2*30+28] {
		t.Fatal("une case jamais atteinte ne doit ni être visible ni explorée")
	}
}
