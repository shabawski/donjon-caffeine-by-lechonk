package game

// EnterBuilding fait entrer le joueur dans le bâtiment adjacent (MMORPG,
// overworld). L'instance de donjon est créée à la demande (une seule par
// bâtiment) puis réutilisée. Retourne true si le joueur a changé de map.
func (s *State) EnterBuilding() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil || m.Kind != MapKindOverworld {
		s.Logf("Pas de bâtiment ici.")
		return false
	}
	b := m.BuildingAdjacent(s.Player.X, s.Player.Y)
	if b == nil {
		s.Logf("Aucun bâtiment à proximité (⌂).")
		return false
	}
	if b.Kind != BuildingDungeonEntrance {
		s.Logf("« %s » n'est pas encore explorable.", b.Name)
		return false
	}
	// Crée l'instance de donjon à la demande (une seule par bâtiment).
	if b.InstanceMapID < 0 {
		inst := s.Maps.Create(MapKindBuilding, 1, m.ID, b.X, b.Y)
		s.populateDungeonMap(inst)
		b.InstanceMapID = inst.ID
		s.Logf("🚪 Tu entres dans le donjon « %s »...", b.Name)
	} else {
		s.Logf("🚪 Tu retournes dans le donjon « %s ».", b.Name)
	}
	s.enterMapInstance(s.Maps.Get(b.InstanceMapID), false)
	return true
}

// ExitBuilding fait sortir le joueur d'une instance de donjon (MMORPG) vers
// la map parente (overworld), à côté du bâtiment. Retourne true si sorti.
func (s *State) ExitBuilding() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil || m.Kind != MapKindBuilding {
		s.Logf("Tu n'es pas dans un donjon.")
		return false
	}
	parent := s.Maps.Get(m.ParentMapID)
	if parent == nil {
		s.Logf("Impossible de sortir d'ici.")
		return false
	}
	s.enterMapInstance(parent, true)
	// Se replacer à côté du bâtiment (case franchissable adjacente).
	for _, b := range parent.Buildings {
		if b.InstanceMapID == m.ID {
			if nx, ny, ok := s.adjacentFree(parent, b.X, b.Y); ok {
				s.Player.X, s.Player.Y = nx, ny
				if fov := parent.FOVFor(s.Player.ID); fov != nil {
					ComputeFOV(parent.World, fov, nx, ny, PlayerViewRange)
				}
			}
			break
		}
	}
	s.Logf("🚪 Tu sors du donjon et retournes au monde ouvert.")
	return true
}

// EnterFirstDungeon entre dans le premier bâtiment donjon de l'overworld
// (téléportation à côté du bâtiment, utile pour le bot et les tests).
// Retourne true si le joueur est entré.
func (s *State) EnterFirstDungeon() bool {
	m := s.curMap()
	if m == nil || m.Kind != MapKindOverworld {
		return false
	}
	for _, b := range m.Buildings {
		if b.Kind != BuildingDungeonEntrance {
			continue
		}
		if nx, ny, ok := s.adjacentFree(m, b.X, b.Y); ok {
			s.Player.X, s.Player.Y = nx, ny
			return s.EnterBuilding()
		}
	}
	return false
}

// adjacentFree trouve une case franchissable et libre (pas de joueur) adjacente
// à (x, y) sur la map donnée (8 voisins, Chebyshev <= 1).
func (s *State) adjacentFree(m *Map, x, y int) (int, int, bool) {
	for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		nx, ny := x+d[0], y+d[1]
		if m.World.IsWalkable(nx, ny) && s.playerAtOnMap(m, nx, ny) == nil {
			return nx, ny, true
		}
	}
	return 0, 0, false
}
