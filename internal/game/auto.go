package game

// AutoAction choisit et applique une action pour le joueur (sur sa map courante).
// Utilisée par le mode --demo (partie automatique headless) et les tests
// d'intégration. Retourne false si la partie est terminée.
//
// Stratégie : farmer chaque étage (tuer les monstres) jusqu'à avoir un niveau
// >= la profondeur, puis descendre. Au dernier étage, chasser le boss.
func (s *State) AutoAction() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil {
		return false
	}
	p := &s.Player
	aliveCount := 0
	m.Entities.EachAlive(func(EntityID) { aliveCount++ })
	// 1. Descendre si sur l'escalier ET (assez farmé OU plus de monstres OU
	//    limite de tours atteinte sur cette map — évite de rester bloqué).
	if m.World.HasStairs && p.X == m.World.StairsX && p.Y == m.World.StairsY &&
		(p.Level >= m.Depth || aliveCount == 0 || m.floorTurns > 300) {
		s.Descend()
		return true
	}
	// 2. Attaquer un monstre adjacent.
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if id := m.Entities.EnemyAt(p.X+d[0], p.Y+d[1]); id != InvalidEntity {
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
	targetX, targetY := m.World.StairsX, m.World.StairsY
	if !m.World.HasStairs {
		if b := s.boss(); b != InvalidEntity {
			targetX, targetY = m.Entities.X[int(b)], m.Entities.Y[int(b)]
		}
	} else if p.Level < m.Depth && aliveCount > 0 {
		if tx, ty, ok := s.nearestEnemyOnMap(m); ok {
			blocked := func(x, y int) bool { return s.Occupied(x, y) }
			if _, _, reachable := BFSNextStep(m.World, blocked, p.X, p.Y, tx, ty); reachable {
				targetX, targetY = tx, ty
			}
		}
	}
	blocked := func(x, y int) bool { return s.Occupied(x, y) }
	if nx, ny, ok := BFSNextStep(m.World, blocked, p.X, p.Y, targetX, targetY); ok && (nx != p.X || ny != p.Y) {
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
// monstres (sur la map courante). Retourne false si aucune fuite n'améliore.
func (s *State) tryFlee() bool {
	p := &s.Player
	curScore := s.minEnemyDist(p.X, p.Y)
	bestX, bestY, bestScore := p.X, p.Y, curScore
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := p.X+d[0], p.Y+d[1]
		if m := s.curMap(); m != nil && m.World.IsWalkable(nx, ny) && !s.Occupied(nx, ny) {
			if score := s.minEnemyDist(nx, ny); score > bestScore {
				bestX, bestY, bestScore = nx, ny, score
			}
		}
	}
	if bestScore > curScore {
		return s.MovePlayer(bestX-p.X, bestY-p.Y)
	}
	return false
}

// minEnemyDist renvoie la distance au monstre vivant le plus proche (map courante).
func (s *State) minEnemyDist(x, y int) int {
	m := s.curMap()
	if m == nil {
		return 1 << 30
	}
	best := 1 << 30
	e := m.Entities
	e.EachAlive(func(id EntityID) {
		i := int(id)
		if d := chebDist(x, y, e.X[i], e.Y[i]); d < best {
			best = d
		}
	})
	return best
}

// nearestEnemyOnMap renvoie le monstre vivant le plus proche sur la map donnée.
func (s *State) nearestEnemyOnMap(m *Map) (int, int, bool) {
	bestDist := 1 << 30
	found := false
	bx, by := 0, 0
	e := m.Entities
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

// AutoPlay joue une partie automatiquement jusqu'à la fin (victoire, défaite
// ou limite de tours) et retourne son résumé textuel.
func AutoPlay(seed int64, maxTurns int) string {
	s := NewState(seed)
	for s.Phase == PhasePlaying && s.Player.Turns < maxTurns {
		s.AutoAction()
	}
	return s.Summary()
}

// AutoPlayMMORPG joue une partie MMORPG automatiquement : entre dans le
// premier donjon de l'overworld, puis joue comme en COOP jusqu'à la fin
// (victoire, défaite ou limite de tours).
func AutoPlayMMORPG(seed int64, maxTurns int) string {
	s := NewStateMMORPG(seed, DefaultWidth, DefaultHeight)
	s.EnterFirstDungeon()
	for s.Phase == PhasePlaying && s.Player.Turns < maxTurns {
		s.AutoAction()
	}
	return s.Summary()
}
