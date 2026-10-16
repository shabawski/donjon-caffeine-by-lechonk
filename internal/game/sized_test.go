package game

import "testing"

func TestClampWorldSize(t *testing.T) {
	tests := []struct {
		w, h         int
		wantW, wantH int
	}{
		{80, 40, 80, 40},            // valeurs normales inchangées
		{5, 3, MinWidth, MinHeight}, // trop petit → borné au minimum
		{20, 10, 20, 10},            // pile au minimum → inchangé
		{200, 100, 200, 100},        // grand → pas de borne haute
	}
	for _, tt := range tests {
		w, h := ClampWorldSize(tt.w, tt.h)
		if w != tt.wantW || h != tt.wantH {
			t.Errorf("ClampWorldSize(%d,%d) = (%d,%d), attendu (%d,%d)",
				tt.w, tt.h, w, h, tt.wantW, tt.wantH)
		}
	}
}

func TestNewStateSized(t *testing.T) {
	s := NewStateSized(42, 60, 30)
	if s.World.Width != 60 || s.World.Height != 30 {
		t.Fatalf("dimensions attendues 60x30, got %dx%d", s.World.Width, s.World.Height)
	}
	if s.FOV == nil || len(s.FOV.Visible) != 60*30 {
		t.Fatalf("FOV mal dimensionné: %d cases", len(s.FOV.Visible))
	}
}

func TestNewStateSizedClampsToMin(t *testing.T) {
	s := NewStateSized(42, 5, 3)
	if s.World.Width != MinWidth || s.World.Height != MinHeight {
		t.Fatalf("dimensions attendues %dx%d, got %dx%d",
			MinWidth, MinHeight, s.World.Width, s.World.Height)
	}
}

func TestNewStateSizedKeepsDimensionsAcrossFloors(t *testing.T) {
	s := NewStateSized(42, 60, 30)
	// Descend à l'étage 2 via l'escalier.
	s.Player.X, s.Player.Y = s.World.StairsX, s.World.StairsY
	if !s.Descend() {
		t.Fatal("la descente doit réussir depuis l'escalier")
	}
	if s.World.Width != 60 || s.World.Height != 30 {
		t.Fatalf("dimensions non conservées à l'étage 2: %dx%d", s.World.Width, s.World.Height)
	}
}
