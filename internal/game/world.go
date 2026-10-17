package game

// Tile est le type d'une case de la carte.
type Tile int

const (
	TileWall Tile = iota
	TileFloor
	TileStairs
	TileStairsUp
)

// Glyph renvoie le caractère représentant la tuile.
func (t Tile) Glyph() rune {
	switch t {
	case TileFloor:
		return '.'
	case TileStairs:
		return '>'
	case TileStairsUp:
		return '<'
	default:
		return '#'
	}
}

const (
	// DefaultWidth et DefaultHeight sont les dimensions d'un étage par défaut.
	// Pensées pour remplir un terminal plein écran (80x40) tout en laissant
	// de la marge pour un viewport plus petit.
	DefaultWidth  = 80
	DefaultHeight = 40
	// MinWidth/MinHeight bornent les dimensions demandées à NewStateSized.
	MinWidth  = 20
	MinHeight = 10
	// MaxDepth est le nombre d'étages du donjon. Le boss garde le dernier.
	MaxDepth = 10
)

// ClampWorldSize borne les dimensions demandées entre MinWidth/MinHeight et
// les valeurs fournies (pas de borne haute : un grand étage reste valide).
func ClampWorldSize(w, h int) (int, int) {
	if w < MinWidth {
		w = MinWidth
	}
	if h < MinHeight {
		h = MinHeight
	}
	return w, h
}

// World est un étage du donjon.
type World struct {
	Width, Height int
	Depth         int // 1..MaxDepth
	Tiles         []Tile
	StairsX       int
	StairsY       int
	HasStairs     bool // escalier vers le bas (false au dernier étage)
	StairsUpX     int
	StairsUpY     int
	HasStairsUp   bool // escalier vers le haut (false au premier étage)
}

// InBounds indique si (x, y) est dans la carte.
func (w *World) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < w.Width && y < w.Height
}

// At renvoie la tuile en (x, y) ; hors limites vaut TileWall.
func (w *World) At(x, y int) Tile {
	if !w.InBounds(x, y) {
		return TileWall
	}
	return w.Tiles[y*w.Width+x]
}

// Set pose la tuile en (x, y) si dans les limites.
func (w *World) Set(x, y int, t Tile) {
	if w.InBounds(x, y) {
		w.Tiles[y*w.Width+x] = t
	}
}

// IsWalkable indique si une case est franchissable (sol ou escalier).
func (w *World) IsWalkable(x, y int) bool {
	switch w.At(x, y) {
	case TileFloor, TileStairs, TileStairsUp:
		return true
	default:
		return false
	}
}

// RandomFloor renvoie une position de sol aléatoire (utile pour les spawns).
func (w *World) RandomFloor(rng *RNG) (x, y int, ok bool) {
	for tries := 0; tries < 200; tries++ {
		x = rng.Intn(w.Width)
		y = rng.Intn(w.Height)
		if w.At(x, y) == TileFloor {
			return x, y, true
		}
	}
	// Repli : scan complet.
	for y = 0; y < w.Height; y++ {
		for x = 0; x < w.Width; x++ {
			if w.At(x, y) == TileFloor {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

type room struct{ x, y, w, h int }

func (r room) center() (int, int) { return r.x + r.w/2, r.y + r.h/2 }

// intersects avec une marge de 1 pour garder au moins un mur entre les salles.
func (r room) intersects(o room) bool {
	return r.x-1 <= o.x+o.w && o.x-1 <= r.x+r.w &&
		r.y-1 <= o.y+o.h && o.y-1 <= r.y+r.h
}

// GenerateWorld crée un étage procédural : des salles disjointes reliées par
// des couloirs en L, plus un escalier vers le bas (absent au dernier étage)
// et un escalier vers le haut (absent au premier étage).
func GenerateWorld(rng *RNG, depth, width, height int) *World {
	w := &World{Width: width, Height: height, Depth: depth, Tiles: make([]Tile, width*height)}
	for i := range w.Tiles {
		w.Tiles[i] = TileWall
	}

	rooms := placeRooms(rng, width, height)
	for i, r := range rooms {
		carveRoom(w, r)
		if i > 0 {
			carveCorridor(w, rng, rooms[i-1], r)
		}
	}

	if depth > 1 {
		// Escalier montant dans la première salle (absent au premier étage).
		first := rooms[0]
		w.StairsUpX, w.StairsUpY = first.center()
		w.Set(w.StairsUpX, w.StairsUpY, TileStairsUp)
		w.HasStairsUp = true
	}
	if depth < MaxDepth {
		last := rooms[len(rooms)-1]
		w.StairsX, w.StairsY = last.center()
		w.Set(w.StairsX, w.StairsY, TileStairs)
		w.HasStairs = true
	}
	return w
}

func placeRooms(rng *RNG, width, height int) []room {
	const maxRooms = 9
	rooms := make([]room, 0, maxRooms)
	for tries := 0; tries < 200 && len(rooms) < maxRooms; tries++ {
		rw := rng.IntRange(4, 9)
		rh := rng.IntRange(3, 7)
		if width-rw-2 < 1 || height-rh-2 < 1 {
			continue
		}
		x := rng.IntRange(1, width-rw-1)
		y := rng.IntRange(1, height-rh-1)
		r := room{x, y, rw, rh}
		ok := true
		for _, o := range rooms {
			if r.intersects(o) {
				ok = false
				break
			}
		}
		if ok {
			rooms = append(rooms, r)
		}
	}
	return rooms
}

func carveRoom(w *World, r room) {
	for y := r.y; y < r.y+r.h; y++ {
		for x := r.x; x < r.x+r.w; x++ {
			w.Set(x, y, TileFloor)
		}
	}
}

func carveCorridor(w *World, rng *RNG, a, b room) {
	ax, ay := a.center()
	bx, by := b.center()
	if rng.Intn(2) == 0 {
		carveH(w, ax, bx, ay)
		carveV(w, ay, by, bx)
	} else {
		carveV(w, ay, by, ax)
		carveH(w, ax, bx, by)
	}
}

func carveH(w *World, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		if w.At(x, y) == TileWall {
			w.Set(x, y, TileFloor)
		}
	}
}

func carveV(w *World, y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		if w.At(x, y) == TileWall {
			w.Set(x, y, TileFloor)
		}
	}
}
