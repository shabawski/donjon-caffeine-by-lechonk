package game

import "testing"

func TestGenerateWorldDimensions(t *testing.T) {
	w := GenerateWorld(NewRNG(1), 1, DefaultWidth, DefaultHeight)
	if w.Width != DefaultWidth || w.Height != DefaultHeight {
		t.Fatalf("dimensions %dx%d, attendu %dx%d", w.Width, w.Height, DefaultWidth, DefaultHeight)
	}
	if len(w.Tiles) != DefaultWidth*DefaultHeight {
		t.Fatalf("len(Tiles)=%d, attendu %d", len(w.Tiles), DefaultWidth*DefaultHeight)
	}
	if w.Depth != 1 {
		t.Fatalf("profondeur %d, attendu 1", w.Depth)
	}
}

func TestGenerateWorldBordersAreWalls(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		w := GenerateWorld(NewRNG(seed), 1, 30, 20)
		for x := 0; x < w.Width; x++ {
			if w.At(x, 0) != TileWall || w.At(x, w.Height-1) != TileWall {
				t.Fatalf("seed %d: bordure horizontale non mur en x=%d", seed, x)
			}
		}
		for y := 0; y < w.Height; y++ {
			if w.At(0, y) != TileWall || w.At(w.Width-1, y) != TileWall {
				t.Fatalf("seed %d: bordure verticale non mur en y=%d", seed, y)
			}
		}
	}
}

func TestGenerateWorldHasFloorAndStairs(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		w := GenerateWorld(NewRNG(seed), 3, DefaultWidth, DefaultHeight)
		floors := 0
		for _, tile := range w.Tiles {
			if tile == TileFloor || tile == TileStairs {
				floors++
			}
		}
		if floors < 20 {
			t.Fatalf("seed %d: trop peu de sol (%d cases)", seed, floors)
		}
		if !w.HasStairs {
			t.Fatalf("seed %d: escalier attendu avant le dernier étage", seed)
		}
		if w.At(w.StairsX, w.StairsY) != TileStairs {
			t.Fatalf("seed %d: case escalier = %v", seed, w.At(w.StairsX, w.StairsY))
		}
		if !w.IsWalkable(w.StairsX, w.StairsY) {
			t.Fatalf("seed %d: l'escalier doit être franchissable", seed)
		}
	}
}

func TestGenerateWorldNoStairsOnLastFloor(t *testing.T) {
	for seed := int64(0); seed < 10; seed++ {
		w := GenerateWorld(NewRNG(seed), MaxDepth, DefaultWidth, DefaultHeight)
		if w.HasStairs {
			t.Fatalf("seed %d: pas d'escalier au dernier étage", seed)
		}
	}
}

func TestGenerateWorldDeterministic(t *testing.T) {
	a := GenerateWorld(NewRNG(99), 2, 40, 20)
	b := GenerateWorld(NewRNG(99), 2, 40, 20)
	for i := range a.Tiles {
		if a.Tiles[i] != b.Tiles[i] {
			t.Fatalf("tuile %d diverge: %v vs %v", i, a.Tiles[i], b.Tiles[i])
		}
	}
	if a.StairsX != b.StairsX || a.StairsY != b.StairsY {
		t.Fatal("position de l'escalier diverge")
	}
}

func TestGenerateWorldFullyConnected(t *testing.T) {
	for seed := int64(0); seed < 10; seed++ {
		w := GenerateWorld(NewRNG(seed), 1, DefaultWidth, DefaultHeight)
		// Trouve la première case franchissable.
		sx, sy, ok := 0, 0, false
		for y := 0; y < w.Height && !ok; y++ {
			for x := 0; x < w.Width && !ok; x++ {
				if w.IsWalkable(x, y) {
					sx, sy, ok = x, y, true
				}
			}
		}
		if !ok {
			t.Fatalf("seed %d: aucune case franchissable", seed)
		}
		reachable := floodFillCount(w, sx, sy)
		total := countWalkable(w)
		if reachable != total {
			t.Fatalf("seed %d: %d cases atteignables sur %d franchissables (carte non connexe)", seed, reachable, total)
		}
	}
}

func TestWorldAtOutOfBounds(t *testing.T) {
	w := &World{Width: 5, Height: 5, Tiles: make([]Tile, 25)}
	if w.At(-1, 0) != TileWall || w.At(0, -1) != TileWall || w.At(5, 0) != TileWall || w.At(0, 5) != TileWall {
		t.Fatal("hors limites doit valoir TileWall")
	}
	if !w.InBounds(0, 0) || !w.InBounds(4, 4) || w.InBounds(5, 5) {
		t.Fatal("InBounds incorrect")
	}
}

func TestTileGlyph(t *testing.T) {
	if TileWall.Glyph() != '#' || TileFloor.Glyph() != '.' || TileStairs.Glyph() != '>' {
		t.Fatal("glyphs de tuiles incorrects")
	}
}
