package game

// newTestState construit un State sur une carte manuelle entourée de murs,
// pour des tests déterministes indépendants de la génération procédurale.
// La carte est enregistrée comme Map (donjon étage 1) dans le registre de maps.
func newTestState(seed int64, w, h int) *State {
	s := &State{RNG: NewRNG(seed), Seed: seed, Phase: PhasePlaying, Maps: NewMapRegistry(seed, w, h), worldWidth: w, worldHeight: h}
	m := s.Maps.Create(MapKindDungeon, 1, NoParentMapID, 0, 0)
	m.World = &World{Width: w, Height: h, Depth: 1, Tiles: make([]Tile, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x == 0 || y == 0 || x == w-1 || y == h-1 {
				m.World.Set(x, y, TileWall)
			} else {
				m.World.Set(x, y, TileFloor)
			}
		}
	}
	s.Player = Player{Entity: *NewPlayer(1, 1), Level: 1, MaxMana: StartMaxMana, Mana: StartMaxMana, MapID: m.ID}
	m.SetFOV(s.Player.ID, NewFOV(w*h))
	ComputeFOV(m.World, m.FOVFor(s.Player.ID), 1, 1, PlayerViewRange)
	return s
}

// addEnemy place un monstre du type donné en (x, y) sur la map courante et
// retourne son EntityID.
func (s *State) addEnemy(kind EntityKind, x, y int) EntityID {
	t := enemyTemplate(kind)
	t.X, t.Y = x, y
	return s.curMap().Entities.Spawn(t)
}

// floodFillCount compte les cases franchissables atteignables depuis (startX, startY).
func floodFillCount(w *World, startX, startY int) int {
	size := w.Width * w.Height
	seen := make([]bool, size)
	start := startY*w.Width + startX
	seen[start] = true
	queue := []int{start}
	count := 0
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		count++
		cx, cy := cur%w.Width, cur/w.Width
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if !w.InBounds(nx, ny) || !w.IsWalkable(nx, ny) || seen[ny*w.Width+nx] {
				continue
			}
			seen[ny*w.Width+nx] = true
			queue = append(queue, ny*w.Width+nx)
		}
	}
	return count
}

// countWalkable compte les cases franchissables de la carte.
func countWalkable(w *World) int {
	n := 0
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			if w.IsWalkable(x, y) {
				n++
			}
		}
	}
	return n
}
