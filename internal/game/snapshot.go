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

// Snapshot capture l'état visible actuel de la partie.
// Le FOV (Visible/Explored) est celui du joueur local ; en multijoueur,
// le serveur construit un snapshot par client avec son propre FOV.
func (s *State) Snapshot() *Snapshot {
	snap := &Snapshot{
		Seed:         s.Seed,
		Phase:        s.Phase,
		Turn:         s.Turn,
		Depth:        s.World.Depth,
		Width:        s.World.Width,
		Height:       s.World.Height,
		Tiles:        s.World.Tiles,
		StairsX:      s.World.StairsX,
		StairsY:      s.World.StairsY,
		HasStairs:    s.World.HasStairs,
		StairsUpX:    s.World.StairsUpX,
		StairsUpY:    s.World.StairsUpY,
		HasStairsUp:  s.World.HasStairsUp,
		Items:        s.Items,
		Visible:      s.FOV.Visible,
		Explored:     s.FOV.Explored,
		Log:          s.Log,
		Won:          s.Won,
		CauseOfDeath: s.CauseOfDeath,
		Player: PlayerSnapshot{
			X: s.Player.X, Y: s.Player.Y, Glyph: s.Player.Glyph,
			Level: s.Player.Level, XP: s.Player.XP, Gold: s.Player.Gold,
			HP: s.Player.HP, MaxHP: s.Player.MaxHP,
			Mana: s.Player.Mana, MaxMana: s.Player.MaxMana,
			Attack: s.Player.Attack, Defense: s.Player.Defense,
			WeaponName: s.Player.WeaponName, WeaponBonus: s.Player.WeaponBonus,
			ArmorName: s.Player.ArmorName, ArmorBonus: s.Player.ArmorBonus,
			EffectiveAtk: s.Player.EffectiveAttack(), EffectiveDef: s.Player.EffectiveDefense(),
			Kills: s.Player.Kills, Turns: s.Player.Turns,
			Inventory: s.Player.Inventory,
		},
		Enemies: s.Entities.AllSnapshots(),
	}
	if s.World.Merchant != nil {
		snap.Merchant = &MerchantSnapshot{
			X: s.World.Merchant.X, Y: s.World.Merchant.Y,
			Glyph: s.World.Merchant.Glyph, Name: s.World.Merchant.Name,
			Shop: s.World.Merchant.Shop,
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
// Le PlayerSnapshot, le FOV (Visible/Explored) et le PlayerID sont ceux de CE
// joueur. Les autres joueurs sont exposés dans Others (rendu multi).
func (s *State) SnapshotForPlayer(p *Player, fov *FOV, playerID int) *Snapshot {
	snap := s.Snapshot()
	snap.PlayerID = playerID
	snap.Player = playerSnapshotOf(p)
	if fov != nil {
		snap.Visible = fov.Visible
		snap.Explored = fov.Explored
	}
	// Autres joueurs : tous les joueurs connectés sauf celui-ci (comparaison par ID).
	snap.Others = make([]PlayerSnapshot, 0, len(s.Others))
	for _, op := range s.AllPlayers() {
		if op.ID == p.ID {
			continue
		}
		snap.Others = append(snap.Others, playerSnapshotOf(op))
	}
	return snap
}
