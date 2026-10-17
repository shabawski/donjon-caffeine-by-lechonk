package game

// AutoAction choisit et applique une action pour le joueur.
// Utilisée par le mode --demo (partie automatique headless) et les tests
// d'intégration. Retourne false si la partie est terminée.
//
// Stratégie : farmer chaque étage (tuer les monstres) jusqu'à avoir un niveau
// >= la profondeur, puis descendre. Au dernier étage, chasser le boss.
func (s *State) AutoAction() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	p := &s.Player
	aliveCount := 0
	s.Entities.EachAlive(func(EntityID) { aliveCount++ })
	// 1. Descendre si sur l'escalier ET (assez farmé OU plus de monstres OU
	//    limite de tours atteinte sur cet étage — évite de rester bloqué).
	if s.World.HasStairs && p.X == s.World.StairsX && p.Y == s.World.StairsY &&
		(p.Level >= s.World.Depth || aliveCount == 0 || s.floorTurns > 300) {
		s.Descend()
		return true
	}
	// 2. Attaquer un monstre adjacent.
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if id := s.OccupiedByEnemy(p.X+d[0], p.Y+d[1]); id != InvalidEntity {
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
	// 5. Cible : boss au dernier étage, monstre le plus proche si pas assez
	//    farmé, sinon l'escalier. Si BFS vers le monstre échoue (mur), on
	//    retombe sur l'escalier.
	targetX, targetY := s.World.StairsX, s.World.StairsY
	if !s.World.HasStairs {
		if b := s.boss(); b != InvalidEntity {
			targetX, targetY = s.Entities.X[int(b)], s.Entities.Y[int(b)]
		}
	} else if p.Level < s.World.Depth && aliveCount > 0 {
		if tx, ty, ok := s.nearestEnemy(); ok {
			blocked := func(x, y int) bool { return s.Occupied(x, y) }
			if _, _, reachable := BFSNextStep(s.World, blocked, p.X, p.Y, tx, ty); reachable {
				targetX, targetY = tx, ty
			}
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

// nearestEnemy renvoie le monstre vivant le plus proche du joueur (pas de
// contrainte de visibilité, contrairement à nearestVisibleEnemy).
func (s *State) nearestEnemy() (int, int, bool) {
	bestDist := 1 << 30
	found := false
	bx, by := 0, 0
	e := s.Entities
	e.EachAlive(func(id EntityID) {
		i := int(id)
		if d := chebDist(s.Player.X, s.Player.Y, e.X[i], e.Y[i]); d < bestDist {
			bestDist = d
			bx, by = e.X[i], e.Y[i]
			found = true
		}
	})
	return bx, by, found
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
	e := s.Entities
	e.EachAlive(func(id EntityID) {
		i := int(id)
		if d := chebDist(x, y, e.X[i], e.Y[i]); d < best {
			best = d
		}
	})
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
