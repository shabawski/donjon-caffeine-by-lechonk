package game

// FOV est le champ de vision d'un étage : cases actuellement visibles et
// cases déjà explorées (fog of war).
type FOV struct {
	Visible  []bool
	Explored []bool
}

// PlayerViewRange est la portée de vue du joueur.
const PlayerViewRange = 8

// NewFOV crée un FOV vierge pour une carte de la taille donnée.
func NewFOV(size int) *FOV {
	return &FOV{Visible: make([]bool, size), Explored: make([]bool, size)}
}

// ComputeFOV calcule la visibilité depuis (x, y) avec la portée donnée.
// Algorithme : un rayon (ligne de Bresenham) est lancé vers chaque case du
// carré de portée ; le rayon s'arrête au premier mur rencontré (inclus).
func ComputeFOV(w *World, fov *FOV, x, y, viewRange int) {
	for i := range fov.Visible {
		fov.Visible[i] = false
	}
	markVisible(w, fov, x, y)
	for dy := -viewRange; dy <= viewRange; dy++ {
		for dx := -viewRange; dx <= viewRange; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			tx, ty := x+dx, y+dy
			if !w.InBounds(tx, ty) || dx*dx+dy*dy > viewRange*viewRange {
				continue
			}
			castRay(w, fov, x, y, tx, ty)
		}
	}
}

// castRay marque visibles les cases de (x0,y0) vers (x1,y1), arrêt au premier mur.
func castRay(w *World, fov *FOV, x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	x, y := x0, y0
	for {
		markVisible(w, fov, x, y)
		if (x == x1 && y == y1) || w.At(x, y) == TileWall {
			return
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func markVisible(w *World, fov *FOV, x, y int) {
	if !w.InBounds(x, y) {
		return
	}
	fov.Visible[y*w.Width+x] = true
	fov.Explored[y*w.Width+x] = true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
