package game

// Génération du monde extérieur (MMORPG).
//
// L'overworld est une carte ouverte (plaine) parsemée de bâtiments (⌂).
// Chaque bâtiment est un portail vers une instance de donjon (créée à la
// demande quand un joueur entre). L'architecture est conçue pour être
// extensible : biomes, PNJ, quêtes, portails vers d'autres overworld maps
// (cartes sibling), etc.

// BuildingKind identifie le type sémantique d'un bâtiment (portail).
type BuildingKind int

const (
	BuildingDungeonEntrance BuildingKind = iota // portail vers un donjon COOP (étages + boss)
	BuildingShop                                // boutique (PNJ marchand en extérieur)
	BuildingInn                                 // auberge (soin, respawn)
)

// Building est un portail/bâtiment sur l'overworld.
type Building struct {
	X, Y  int
	Glyph rune
	Kind  BuildingKind
	Name  string
	// InstanceMapID : map instance de donjon liée à ce bâtiment (-1 = pas
	// encore créée). Créée à la demande quand un joueur entre (MapRegistry),
	// réutilisée ensuite — une seule instance par bâtiment.
	InstanceMapID int
}

// TileBuilding est déclarée dans world.go (bloc iota des tuiles) : c'est un
// bâtiment (portail, infranchissable, interagissable avec 'g').

// GenerateOverworld génère une carte de monde ouvert : une plaine ouverte
// (pas de salles fermées) avec quelques bâtiments (⌂) placés sur des cases
// floor. Les bâtiments sont des portails vers des instances de donjon.
//
// Architecture extensible : cette fonction peut être remplacée/enrichie
// (biomes, routes, PNJ, portails inter-maps) sans toucher au reste du moteur.
func GenerateOverworld(m *Map, rng *RNG) {
	w := m.World
	// Tout en floor (monde ouvert), bordures en murs.
	for i := range w.Tiles {
		w.Tiles[i] = TileFloor
	}
	for x := 0; x < w.Width; x++ {
		w.Set(x, 0, TileWall)
		w.Set(x, w.Height-1, TileWall)
	}
	for y := 0; y < w.Height; y++ {
		w.Set(0, y, TileWall)
		w.Set(w.Width-1, y, TileWall)
	}
	// Pas d'escaliers en overworld (c'est un monde plat, pas un donjon).
	w.HasStairs = false
	w.HasStairsUp = false

	// Quelques éléments de décor : arbres (murs partiels) pour le relief.
	placeOverworldDecor(w, rng, 15)

	// Place 2-3 bâtiments (portails vers donjons).
	buildings := placeBuildings(w, rng, 3)
	m.Buildings = buildings
}

// placeOverworldDecor place des éléments de décor (arbres = murs) pour le relief.
func placeOverworldDecor(w *World, rng *RNG, count int) {
	for i := 0; i < count; i++ {
		x, y := rng.Intn(w.Width-2)+1, rng.Intn(w.Height-2)+1
		if w.At(x, y) == TileFloor {
			w.Set(x, y, TileWall) // arbre
		}
	}
}

// placeBuildings place N bâtiments (⌂) sur des cases floor libres.
func placeBuildings(w *World, rng *RNG, n int) []*Building {
	buildings := make([]*Building, 0, n)
	kinds := []BuildingKind{BuildingDungeonEntrance, BuildingShop, BuildingInn}
	names := []string{"Donjon", "Boutique", "Auberge"}
	for i := 0; i < n; i++ {
		for tries := 0; tries < 50; tries++ {
			x, y := rng.Intn(w.Width-2)+1, rng.Intn(w.Height-2)+1
			if w.At(x, y) != TileFloor {
				continue
			}
			kind := kinds[i%len(kinds)]
			b := &Building{
				X: x, Y: y, Glyph: '⌂', Kind: kind, Name: names[i%len(names)],
				InstanceMapID: -1, // instance de donjon créée à la demande
			}
			w.Set(x, y, TileBuilding)
			buildings = append(buildings, b)
			break
		}
	}
	return buildings
}
