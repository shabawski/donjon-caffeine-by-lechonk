package game

// Snapshot est une capture sérialisable (JSON) de l'état visible d'une partie.
// C'est le seul état que le client TUI reçoit du serveur : le TUI rend depuis
// un Snapshot, jamais directement depuis un *State (qui reste côté serveur).
//
// En solo, le serveur tourne dans le même process (channels) ; en multi,
// chaque client reçoit le snapshot avec SON champ de vision (FOV par joueur).
type Snapshot struct {
	Seed         int64             `json:"seed"`
	Phase        Phase             `json:"phase"`
	Mode         GameMode          `json:"mode"` // COOP (donjon) ou MMORPG (monde ouvert)
	Turn         int               `json:"turn"`
	Depth        int               `json:"depth"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	Tiles        []Tile            `json:"tiles"`
	StairsX      int               `json:"stairsX"`
	StairsY      int               `json:"stairsY"`
	HasStairs    bool              `json:"hasStairs"`
	StairsUpX    int               `json:"stairsUpX"`
	StairsUpY    int               `json:"stairsUpY"`
	HasStairsUp  bool              `json:"hasStairsUp"`
	Merchant     *MerchantSnapshot `json:"merchant,omitempty"`
	Player       PlayerSnapshot    `json:"player"`
	Enemies      []Entity          `json:"enemies"` // snapshots des entités vivantes
	Items        []*Item           `json:"items"`
	Visible      []bool            `json:"visible"`  // FOV du joueur (cases visibles)
	Explored     []bool            `json:"explored"` // cases déjà explorées (fog of war)
	Log          []string          `json:"log"`
	Won          bool              `json:"won"`
	CauseOfDeath string            `json:"causeOfDeath"`
	PlayerID     int               `json:"playerId"`         // joueur concerné par ce snapshot (0 = solo)
	Others       []PlayerSnapshot  `json:"others,omitempty"` // autres joueurs visibles (multi)
}

// PlayerSnapshot est la partie visible du joueur dans un Snapshot.
type PlayerSnapshot struct {
	X            int     `json:"x"`
	Y            int     `json:"y"`
	Glyph        rune    `json:"glyph"`
	Level        int     `json:"level"`
	XP           int     `json:"xp"`
	Gold         int     `json:"gold"`
	HP           int     `json:"hp"`
	MaxHP        int     `json:"maxHp"`
	Mana         int     `json:"mana"`
	MaxMana      int     `json:"maxMana"`
	Attack       int     `json:"attack"`
	Defense      int     `json:"defense"`
	WeaponName   string  `json:"weaponName"`
	WeaponBonus  int     `json:"weaponBonus"`
	ArmorName    string  `json:"armorName"`
	ArmorBonus   int     `json:"armorBonus"`
	EffectiveAtk int     `json:"effectiveAtk"`
	EffectiveDef int     `json:"effectiveDef"`
	Kills        int     `json:"kills"`
	Turns        int     `json:"turns"`
	Inventory    []*Item `json:"inventory"`
}

// MerchantSnapshot est la partie visible du marchand dans un Snapshot.
type MerchantSnapshot struct {
	X     int        `json:"x"`
	Y     int        `json:"y"`
	Glyph rune       `json:"glyph"`
	Name  string     `json:"name"`
	Shop  []ShopItem `json:"shop"`
}

// Snapshot capture l'état visible de la map courante du joueur local.
// Le FOV (Visible/Explored) est celui du joueur local dans sa map.
func (s *State) Snapshot() *Snapshot {
	m := s.curMap()
	if m == nil {
		return &Snapshot{Seed: s.Seed, Phase: s.Phase}
	}
	fov := s.curFOV()
	snap := &Snapshot{
		Seed:         s.Seed,
		Phase:        s.Phase,
		Mode:         s.Mode,
		Turn:         s.Turn,
		Depth:        m.Depth,
		Width:        m.World.Width,
		Height:       m.World.Height,
		Tiles:        m.World.Tiles,
		StairsX:      m.World.StairsX,
		StairsY:      m.World.StairsY,
		HasStairs:    m.World.HasStairs,
		StairsUpX:    m.World.StairsUpX,
		StairsUpY:    m.World.StairsUpY,
		HasStairsUp:  m.World.HasStairsUp,
		Items:        m.Items,
		Log:          s.Log,
		Won:          s.Won,
		CauseOfDeath: s.CauseOfDeath,
		Player:       playerSnapshotOf(&s.Player),
		Enemies:      m.Entities.AllSnapshots(),
	}
	if fov != nil {
		snap.Visible = fov.Visible
		snap.Explored = fov.Explored
	}
	if m.Merchant != nil {
		snap.Merchant = &MerchantSnapshot{
			X: m.Merchant.X, Y: m.Merchant.Y,
			Glyph: m.Merchant.Glyph, Name: m.Merchant.Name,
			Shop: m.Merchant.Shop,
		}
	}
	return snap
}

// InBounds indique si (x, y) est dans la carte.
func (s *Snapshot) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < s.Width && y < s.Height
}

// At renvoie la tuile en (x, y) ; hors limites vaut TileWall.
func (s *Snapshot) At(x, y int) Tile {
	if !s.InBounds(x, y) {
		return TileWall
	}
	return s.Tiles[y*s.Width+x]
}

// IsWalkable indique si une case est franchissable.
func (s *Snapshot) IsWalkable(x, y int) bool {
	switch s.At(x, y) {
	case TileFloor, TileStairs, TileStairsUp:
		return true
	default:
		return false
	}
}

// EnemyAt renvoie le snapshot du monstre en (x, y), ou nil.
func (s *Snapshot) EnemyAt(x, y int) *Entity {
	for i := range s.Enemies {
		if s.Enemies[i].X == x && s.Enemies[i].Y == y {
			return &s.Enemies[i]
		}
	}
	return nil
}

// ItemAt renvoie l'objet posé en (x, y), ou nil.
func (s *Snapshot) ItemAt(x, y int) *Item {
	for _, it := range s.Items {
		if it.X == x && it.Y == y {
			return it
		}
	}
	return nil
}

// Score calcule le score de la partie (même formule que State.Score).
func (s *Snapshot) Score() int {
	score := s.Player.Gold*2 + s.Player.Kills*10 + s.Player.Level*25 + s.Depth*50
	if s.Won {
		score += 1000
	}
	return score
}

// playerSnapshotOf construit le PlayerSnapshot d'un joueur.
func playerSnapshotOf(p *Player) PlayerSnapshot {
	return PlayerSnapshot{
		X: p.X, Y: p.Y, Glyph: p.Glyph,
		Level: p.Level, XP: p.XP, Gold: p.Gold,
		HP: p.HP, MaxHP: p.MaxHP,
		Mana: p.Mana, MaxMana: p.MaxMana,
		Attack: p.Attack, Defense: p.Defense,
		WeaponName: p.WeaponName, WeaponBonus: p.WeaponBonus,
		ArmorName: p.ArmorName, ArmorBonus: p.ArmorBonus,
		EffectiveAtk: p.EffectiveAttack(), EffectiveDef: p.EffectiveDefense(),
		Kills: p.Kills, Turns: p.Turns,
		Inventory: p.Inventory,
	}
}

// SnapshotForPlayer construit un snapshot pour un joueur spécifique (multi).
// Le snapshot reflète la MAP DU JOUEUR (pas la map courante du serveur) :
// tuiles, entités, items, marchand de SA map, et son FOV personnel.
// Les autres joueurs exposés dans Others sont ceux sur la MÊME map.
func (s *State) SnapshotForPlayer(p *Player, fov *FOV, playerID int) *Snapshot {
	m := s.Maps.Get(p.MapID)
	if m == nil {
		m = s.curMap()
	}
	if m == nil {
		return &Snapshot{Seed: s.Seed, Phase: s.Phase}
	}
	snap := &Snapshot{
		Seed:         s.Seed,
		Phase:        s.Phase,
		Mode:         s.Mode,
		Turn:         s.Turn,
		Depth:        m.Depth,
		Width:        m.World.Width,
		Height:       m.World.Height,
		Tiles:        m.World.Tiles,
		StairsX:      m.World.StairsX,
		StairsY:      m.World.StairsY,
		HasStairs:    m.World.HasStairs,
		StairsUpX:    m.World.StairsUpX,
		StairsUpY:    m.World.StairsUpY,
		HasStairsUp:  m.World.HasStairsUp,
		Items:        m.Items,
		Log:          s.Log,
		Won:          s.Won,
		CauseOfDeath: s.CauseOfDeath,
		PlayerID:     playerID,
		Player:       playerSnapshotOf(p),
		Enemies:      m.Entities.AllSnapshots(),
	}
	if fov != nil {
		snap.Visible = fov.Visible
		snap.Explored = fov.Explored
	}
	if m.Merchant != nil {
		snap.Merchant = &MerchantSnapshot{
			X: m.Merchant.X, Y: m.Merchant.Y,
			Glyph: m.Merchant.Glyph, Name: m.Merchant.Name,
			Shop: m.Merchant.Shop,
		}
	}
	// Autres joueurs sur la MÊME map (comparaison par ID).
	snap.Others = make([]PlayerSnapshot, 0)
	for _, op := range s.playersOnMap(m.ID) {
		if op.ID == p.ID {
			continue
		}
		snap.Others = append(snap.Others, playerSnapshotOf(op))
	}
	return snap
}
