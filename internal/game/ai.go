package game

// BFSNextStep trouve le prochain pas d'un chemin BFS de (fromX,fromY) vers
// (toX,toY) sur les cases franchissables. blocked permet d'exclure des cases
// (ex: cases occupées par d'autres monstres). Retourne ok=false sans chemin.
func BFSNextStep(w *World, blocked func(x, y int) bool, fromX, fromY, toX, toY int) (nx, ny int, ok bool) {
	if !w.InBounds(fromX, fromY) || !w.InBounds(toX, toY) {
		return 0, 0, false
	}
	if fromX == toX && fromY == toY {
		return fromX, fromY, true
	}
	size := w.Width * w.Height
	dist := make([]int, size)
	for i := range dist {
		dist[i] = -1
	}
	type node struct{ x, y int }
	queue := make([]node, 0, size)
	dist[fromY*w.Width+fromX] = 0
	queue = append(queue, node{fromX, fromY})
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	found := false
	for head := 0; head < len(queue) && !found; head++ {
		cur := queue[head]
		for _, d := range dirs {
			cx, cy := cur.x+d[0], cur.y+d[1]
			if !w.InBounds(cx, cy) || !w.IsWalkable(cx, cy) || dist[cy*w.Width+cx] != -1 {
				continue
			}
			if blocked != nil && blocked(cx, cy) {
				continue
			}
			dist[cy*w.Width+cx] = dist[cur.y*w.Width+cur.x] + 1
			if cx == toX && cy == toY {
				found = true
				break
			}
			queue = append(queue, node{cx, cy})
		}
	}
	if !found {
		return 0, 0, false
	}
	// Remonte depuis l'arrivée en suivant les distances décroissantes ;
	// la case atteinte à distance 1 est le prochain pas depuis le départ.
	cx, cy := toX, toY
	for dist[cy*w.Width+cx] > 1 {
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if !w.InBounds(nx, ny) {
				continue
			}
			if dist[ny*w.Width+nx] == dist[cy*w.Width+cx]-1 {
				cx, cy = nx, ny
				break
			}
		}
	}
	return cx, cy, true
}

// EnemyTurn fait agir un monstre pour ce tour : patrouille si le joueur n'est
// pas détecté, fuite si le monstre est faible et blessé, chasse (BFS) sinon,
// attaque au corps à corps si adjacent.
// Le monstre est identifié par son EntityID dans le registre SOA s.Entities.
func (s *State) EnemyTurn(id EntityID) {
	e := s.Entities
	if !e.IsAlive(id) {
		return
	}
	i := int(id)
	px, py := s.Player.X, s.Player.Y
	dist := chebDist(e.X[i], e.Y[i], px, py)
	playerVisible := s.FOV.Visible[py*s.World.Width+px]
	detected := playerVisible && dist <= e.DetectRange[i]
	if e.Kind[i] == KindBoss && dist <= e.DetectRange[i] {
		detected = true // le boss flaire le café du joueur de loin
	}
	if !detected {
		// Patrouille : un déplacement sur deux vers une case voisine au hasard.
		if s.RNG.Chance(0.5) {
			s.moveEnemyRandom(id)
		}
		return
	}
	if e.FleeBelow[i] > 0 && float64(e.HP[i])/float64(e.MaxHP[i]) < e.FleeBelow[i] {
		s.moveEnemyAway(id, px, py)
		return
	}
	if dist == 1 {
		atk := e.Attack[i]
		if e.Kind[i] == KindBoss && float64(e.HP[i])/float64(e.MaxHP[i]) < 0.5 {
			atk += 2 // le boss se met en colère quand il est blessé
		}
		def := s.Player.EffectiveDefense()
		dmg, crit, killed := Attack(s.RNG, atk, def, &s.Player.HP)
		if crit {
			s.Logf("💥 Le %s te porte un coup critique ! -%d PV (%d/%d)", e.Name[i], dmg, s.Player.HP, s.Player.MaxHP)
		} else {
			s.Logf("Le %s t'attaque : -%d PV (%d/%d)", e.Name[i], dmg, s.Player.HP, s.Player.MaxHP)
		}
		if killed {
			s.CauseOfDeath = "tué par un(e) " + e.Name[i]
			s.PlayerDeath()
		}
		return
	}
	blocked := func(x, y int) bool { return s.OccupiedByEnemy(x, y) != InvalidEntity }
	nx, ny, ok := BFSNextStep(s.World, blocked, e.X[i], e.Y[i], px, py)
	if !ok {
		s.moveEnemyRandom(id)
		return
	}
	s.moveEnemyTo(id, nx, ny)
}

// moveEnemyRandom déplace le monstre vers une case voisine franchissable au hasard.
func (s *State) moveEnemyRandom(id EntityID) {
	e := s.Entities
	i := int(id)
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	s.RNG.Shuffle(4, func(a, b int) { dirs[a], dirs[b] = dirs[b], dirs[a] })
	for _, d := range dirs {
		if s.CanEnemyMoveTo(e.X[i]+d[0], e.Y[i]+d[1]) {
			s.moveEnemyTo(id, e.X[i]+d[0], e.Y[i]+d[1])
			return
		}
	}
}

// moveEnemyAway déplace le monstre vers la case voisine qui l'éloigne le plus du joueur.
func (s *State) moveEnemyAway(id EntityID, px, py int) {
	e := s.Entities
	i := int(id)
	type cand struct{ x, y, score int }
	best := cand{score: -1}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := e.X[i]+dx, e.Y[i]+dy
			if !s.CanEnemyMoveTo(nx, ny) {
				continue
			}
			if score := chebDist(nx, ny, px, py); score > best.score {
				best = cand{nx, ny, score}
			}
		}
	}
	if best.score >= 0 {
		s.moveEnemyTo(id, best.x, best.y)
	}
}

func (s *State) moveEnemyTo(id EntityID, x, y int) {
	i := int(id)
	s.Entities.X[i] = x
	s.Entities.Y[i] = y
}

// chebDist est la distance de Chebyshev (max des distances sur chaque axe).
func chebDist(x1, y1, x2, y2 int) int {
	dx := abs(x1 - x2)
	dy := abs(y1 - y2)
	if dx > dy {
		return dx
	}
	return dy
}
