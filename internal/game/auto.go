package game

// AutoAction choisit et applique une action pour le joueur.
// Utilisée par le mode --demo (partie automatique headless) et les tests
// d'intégration. Retourne false si la partie est terminée.
func (s *State) AutoAction() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	p := &s.Player
	// 1. Descendre si le joueur est sur l'escalier.
	if s.World.HasStairs && p.X == s.World.StairsX && p.Y == s.World.StairsY {
		s.Descend()
		return true
	}
	// 2. Attaquer un monstre adjacent.
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if e := s.OccupiedByEnemy(p.X+d[0], p.Y+d[1]); e != nil {
			s.MovePlayer(d[0], d[1])
			return true
		}
	}
	// 3. Boule de feu si assez de mana et un monstre visible à portée.
	if p.Mana >= FireballCost {
		if tx, ty, ok := s.nearestVisibleEnemy(); ok && chebDist(p.X, p.Y, tx, ty) <= 6 {
			s.CastFireball(0, 0)
			return true
		}
	}
	// 4. Fuir si les PV sont bas.
	if float64(p.HP)/float64(p.MaxHP) < 0.35 && s.tryFlee() {
		return true
	}
	// 5. Se diriger vers l'escalier (ou chasser le boss au dernier étage).
	targetX, targetY := s.World.StairsX, s.World.StairsY
	if !s.World.HasStairs {
		if b := s.boss(); b != nil {
			targetX, targetY = b.X, b.Y
		}
	}
	blocked := func(x, y int) bool { return s.Occupied(x, y) }
	if nx, ny, ok := BFSNextStep(s.World, blocked, p.X, p.Y, targetX, targetY); ok && (nx != p.X || ny != p.Y) {
		s.MovePlayer(nx-p.X, ny-p.Y)
		return true
	}
	// 6. Errance aléatoire en dernier recours.
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	s.RNG.Shuffle(4, func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
	for _, d := range dirs {
		if s.MovePlayer(d[0], d[1]) {
			return true
		}
	}
	s.PlayerWait()
	return true
}

// tryFlee déplace le joueur vers la case voisine qui l'éloigne le plus des
// monstres. Retourne false si aucune fuite n'améliore la situation.
func (s *State) tryFlee() bool {
	p := &s.Player
	curScore := s.minEnemyDist(p.X, p.Y)
	bestX, bestY, bestScore := p.X, p.Y, curScore
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := p.X+d[0], p.Y+d[1]
		if !s.World.IsWalkable(nx, ny) || s.Occupied(nx, ny) {
			continue
		}
		if score := s.minEnemyDist(nx, ny); score > bestScore {
			bestX, bestY, bestScore = nx, ny, score
		}
	}
	if bestScore > curScore {
		return s.MovePlayer(bestX-p.X, bestY-p.Y)
	}
	return false
}

// minEnemyDist renvoie la distance au monstre vivant le plus proche.
func (s *State) minEnemyDist(x, y int) int {
	best := 1 << 30
	for _, e := range s.Enemies {
		if !e.IsAlive() {
			continue
		}
		if d := chebDist(x, y, e.X, e.Y); d < best {
			best = d
		}
	}
	return best
}

// AutoPlay joue une partie automatiquement jusqu'à la fin (victoire, défaite
// ou limite de tours) et retourne son résumé textuel.
func AutoPlay(seed int64, maxTurns int) string {
	s := NewState(seed)
	for s.Phase == PhasePlaying && s.Player.Turns < maxTurns {
		s.AutoAction()
	}
	return s.Summary()
}
