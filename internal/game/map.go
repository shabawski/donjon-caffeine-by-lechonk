package game

// Map est une instance d'étage : un monde complet avec ses entités, items,
// marchand, et les FOV des joueurs présents. Plusieurs maps coexistent dans
// une partie (State.Maps) — c'est la base de l'architecture multi-maps :
//
//   - Un étage du donjon = une Map (profondeur = Depth).
//   - Le monde extérieur (MMORPG) = une Map spéciale (Depth = 0, Kind = MapKindOverworld).
//   - Un bâtiment/portail vers un donjon = une Map instance, reliée à la map
//     parente par ParentMapID + EntranceX/Y.
//
// Chaque joueur a un MapID : il ne voit et n'agit que dans sa map courante.
// Descendre/monter = changer de MapID (la map parente persiste, les autres
// joueurs sur les autres maps ne sont pas affectés).
type Map struct {
	ID          int
	Kind        MapKind
	Depth       int // 0 = extérieur/ville (overworld), 1..MaxDepth = donjon
	ParentMapID int // map parente (0 si racine) — pour remonter
	EntranceX   int // point d'apparition en arrivant de la map parente
	EntranceY   int
	World       *World
	Entities    *Entities    // monstres de CETTE map (registre ECS/SOA)
	Items       []*Item      // objets au sol de CETTE map
	Merchant    *Merchant    // marchand de CETTE map (nil si absent)
	Buildings   []*Building  // bâtiments/portails de CETTE map (overworld, nil sinon)
	FOVs        map[int]*FOV // playerID → FOV dans cette map (multi)
	floorTurns  int          // tours passés sur cette map (bot farm)
}

// MapKind distingue le type sémantique d'une map.
type MapKind int

const (
	MapKindDungeon   MapKind = iota // étage de donjon (salles + couloirs + escaliers)
	MapKindOverworld                // monde extérieur (MMORPG : carte sibling, bâtiments)
	MapKindBuilding                 // instance de bâtiment (portail vers donjon)
)

// NewMap crée une map vide (le World est généré par le caller via GenerateWorld
// ou GenerateOverworld selon le Kind).
func NewMap(id int, kind MapKind, depth, parentMapID, entranceX, entranceY, width, height int) *Map {
	return &Map{
		ID:          id,
		Kind:        kind,
		Depth:       depth,
		ParentMapID: parentMapID,
		EntranceX:   entranceX,
		EntranceY:   entranceY,
		World:       &World{Width: width, Height: height, Depth: depth, Tiles: make([]Tile, width*height)},
		Entities:    NewEntities(),
		Items:       nil,
		FOVs:        make(map[int]*FOV),
	}
}

// PlayerIDs retourne les IDs des joueurs présents sur cette map.
// (La liste des joueurs est tenue par le State/serveur ; la map ne stocke que
// les FOV. Cette méthode est un helper pour le serveur.)
func (m *Map) FOVFor(playerID int) *FOV {
	if fov, ok := m.FOVs[playerID]; ok {
		return fov
	}
	return nil
}

// SetFOV attache ou remplace le FOV d'un joueur sur cette map.
func (m *Map) SetFOV(playerID int, fov *FOV) {
	m.FOVs[playerID] = fov
}

// BuildingAt retourne le bâtiment à la case (x, y), ou nil.
func (m *Map) BuildingAt(x, y int) *Building {
	for _, b := range m.Buildings {
		if b.X == x && b.Y == y {
			return b
		}
	}
	return nil
}

// BuildingAdjacent retourne le bâtiment à distance de Chebyshev <= 1 de
// (x, y) (case du joueur ou case voisine), ou nil.
func (m *Map) BuildingAdjacent(x, y int) *Building {
	for _, b := range m.Buildings {
		if chebDist(x, y, b.X, b.Y) <= 1 {
			return b
		}
	}
	return nil
}

// MapRegistry gère toutes les maps vivantes d'une partie. Le nombre de maps
// est illimité (création à la demande quand un joueur entre dans un étage).
type MapRegistry struct {
	maps   map[int]*Map
	nextID int
	seed   int64
	width  int
	height int
}

// NewMapRegistry crée un registre vide (les dimensions par défaut pour les
// nouvelles maps).
func NewMapRegistry(seed int64, width, height int) *MapRegistry {
	return &MapRegistry{
		maps:   make(map[int]*Map),
		seed:   seed,
		width:  width,
		height: height,
	}
}

// Get retourne la map par ID, ou nil.
func (r *MapRegistry) Get(id int) *Map {
	return r.maps[id]
}

// All retourne toutes les maps vivantes.
func (r *MapRegistry) All() []*Map {
	out := make([]*Map, 0, len(r.maps))
	for _, m := range r.maps {
		out = append(out, m)
	}
	return out
}

// Len retourne le nombre de maps vivantes.
func (r *MapRegistry) Len() int {
	return len(r.maps)
}

// Create crée et enregistre une nouvelle map (ID auto-incrémenté).
func (r *MapRegistry) Create(kind MapKind, depth, parentMapID, entranceX, entranceY int) *Map {
	id := r.nextID
	r.nextID++
	m := NewMap(id, kind, depth, parentMapID, entranceX, entranceY, r.width, r.height)
	r.maps[id] = m
	return m
}

// Remove supprime une map du registre (si plus aucun joueur dessus).
func (r *MapRegistry) Remove(id int) {
	delete(r.maps, id)
}

// MapAt retourne la map à la profondeur donnée pour le parent donné
// (une seule map par (parent, depth) — les joueurs d'un même étage partagent
// la même instance en COOP). Retourne nil si aucune.
func (r *MapRegistry) MapAt(parentMapID, depth int) *Map {
	for _, m := range r.maps {
		if m.ParentMapID == parentMapID && m.Depth == depth {
			return m
		}
	}
	return nil
}

// Root retourne la map racine (sans parente), ou nil.
func (r *MapRegistry) Root() *Map {
	for _, m := range r.maps {
		if m.ParentMapID == NoParentMapID {
			return m
		}
	}
	return nil
}
