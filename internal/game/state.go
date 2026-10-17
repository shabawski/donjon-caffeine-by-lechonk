package game

import "fmt"

// Phase est la phase courante du jeu.
type Phase int

const (
	PhaseMenu Phase = iota
	PhasePlaying
	PhaseGameOver
	PhaseVictory
)

const (
	StartMaxMana   = 20
	FireballCost   = 8 // mana dépensé par boule de feu
	FireballBase   = 8 // dégâts de base (+2 par niveau)
	ManaRegenEvery = 3 // +1 mana tous les 3 tours
	MaxLog         = 200
)

// GameMode identifie le mode de jeu (COOP donjon ou MMORPG monde ouvert).
type GameMode int

const (
	ModeCoop   GameMode = iota // donjon : tuer le boss qui a volé le café
	ModeMMORPG                 // monde ouvert : overworld + instances de donjon
)

// Player est le personnage contrôlé par le joueur.
// Reste une struct à part entière (pas une entité ECS). Chaque joueur a un
// MapID : la map (instance d'étage) dans laquelle il se trouve.
type Player struct {
	Entity
	ID          int // identifiant unique (0 = solo, >0 en multi)
	MapID       int // map courante du joueur (étage du donjon, ou overworld)
	Level       int
	XP          int
	Gold        int
	WeaponName  string
	WeaponBonus int
	ArmorName   string
	ArmorBonus  int
	MaxMana     int
	Mana        int
	Turns       int
	Kills       int
	Inventory   []*Item // objets gardés sur soi (armes, armures, potions)
}

// State est l'état global d'une partie : registre de maps + état global.
//
// Architecture v3 (maps) : un étage = une Map (instance). Plusieurs maps
// coexistent (State.Maps) — les joueurs sont répartis par MapID. Quand un
// joueur descend/monte, il change de MapID : les autres joueurs sur les
// autres maps ne sont PAS affectés (fix bug 17/10 17:07).
//
// Le joueur local est State.Player ; en multi, les autres sont State.Others
// (le serveur swap Player/Others à chaque input, comme en v2).
type State struct {
	RNG          *RNG
	Seed         int64
	Phase        Phase
	Mode         GameMode
	Maps         *MapRegistry // registre de toutes les maps vivantes
	Player       Player       // joueur local (solo)
	Others       []*Player    // autres joueurs connectés (vide en solo, rempli en multi)
	Log          []string
	Turn         int
	Won          bool
	CauseOfDeath string
	worldWidth   int // dimensions par défaut pour les nouvelles maps
	worldHeight  int
}

// curMap retourne la map du joueur local.
func (s *State) curMap() *Map {
	return s.Maps.Get(s.Player.MapID)
}

// curFOV retourne le FOV du joueur local dans sa map courante (le crée si absent).
func (s *State) curFOV() *FOV {
	m := s.curMap()
	if m == nil {
		return nil
	}
	if fov, ok := m.FOVs[s.Player.ID]; ok {
		return fov
	}
	fov := NewFOV(m.World.Width * m.World.Height)
	m.FOVs[s.Player.ID] = fov
	return fov
}

// CurrentMap retourne la map courante du joueur local (exporté pour le client/TUI).
func (s *State) CurrentMap() *Map { return s.curMap() }

// CurrentFOV retourne le FOV du joueur local dans sa map courante (exporté pour le client/TUI).
func (s *State) CurrentFOV() *FOV { return s.curFOV() }

// AllPlayers retourne tous les joueurs connectés (local + autres).
// Utilisé par l'IA pour le focus (le monstre cible le joueur le plus proche).
func (s *State) AllPlayers() []*Player {
	players := make([]*Player, 0, 1+len(s.Others))
	players = append(players, &s.Player)
	players = append(players, s.Others...)
	return players
}

// playersOnCurMap retourne les joueurs présents sur la map courante.
// En solo : juste le joueur local. En multi : local + autres sur la même map.
func (s *State) playersOnCurMap() []*Player {
	m := s.curMap()
	if m == nil {
		return s.AllPlayers()
	}
	return s.playersOnMap(m.ID)
}

// playersOnMap retourne les joueurs présents sur la map donnée.
func (s *State) playersOnMap(mapID int) []*Player {
	var out []*Player
	for _, p := range s.AllPlayers() {
		if p.MapID == mapID {
			out = append(out, p)
		}
	}
	return out
}

// playerAt retourne le joueur (sur la map courante) à la position (x, y), ou nil.
func (s *State) playerAt(x, y int) *Player {
	for _, p := range s.playersOnCurMap() {
		if p.X == x && p.Y == y {
			return p
		}
	}
	return nil
}

// NewState crée une nouvelle partie COOP (donjon) avec la graine donnée.
func NewState(seed int64) *State {
	return NewStateSized(seed, DefaultWidth, DefaultHeight)
}

// NewStateSized crée une nouvelle partie COOP aux dimensions données.
// La map racine (étage 1) est créée et peuplée.
func NewStateSized(seed int64, width, height int) *State {
	width, height = ClampWorldSize(width, height)
	s := &State{
		RNG:         NewRNG(seed),
		Seed:        seed,
		Phase:       PhasePlaying,
		Mode:        ModeCoop,
		Maps:        NewMapRegistry(seed, width, height),
		worldWidth:  width,
		worldHeight: height,
	}
	s.Player = Player{
		Entity:  *NewPlayer(0, 0),
		Level:   1,
		MaxMana: StartMaxMana,
		Mana:    StartMaxMana,
	}
	s.enterMap(1) // map racine : étage 1 du donjon
	return s
}

// NewStateMMORPG crée une nouvelle partie MMORPG : la map racine est
// l'overworld (monde extérieur, Depth=0, Kind=MapKindOverworld).
func NewStateMMORPG(seed int64, width, height int) *State {
	width, height = ClampWorldSize(width, height)
	s := &State{
		RNG:         NewRNG(seed),
		Seed:        seed,
		Phase:       PhasePlaying,
		Mode:        ModeMMORPG,
		Maps:        NewMapRegistry(seed, width, height),
		worldWidth:  width,
		worldHeight: height,
	}
	s.Player = Player{
		Entity:  *NewPlayer(0, 0),
		Level:   1,
		MaxMana: StartMaxMana,
		Mana:    StartMaxMana,
	}
	// Map racine : overworld (monde extérieur).
	m := s.Maps.Create(MapKindOverworld, 0, NoParentMapID, 0, 0)
	GenerateOverworld(m, s.RNG)
	px, py, _ := m.World.RandomFloor(s.RNG)
	s.Player.X, s.Player.Y = px, py
	s.Player.MapID = m.ID
	m.SetFOV(s.Player.ID, NewFOV(m.World.Width*m.World.Height))
	ComputeFOV(m.World, m.FOVFor(s.Player.ID), px, py, PlayerViewRange)
	s.Logf("🌍 Bienvenue dans le monde ouvert ! Explore, entre dans les bâtiments (⌂).")
	return s
}

// NoParentMapID marque une map racine (sans parente) : -1, car 0 est un ID valide.
const NoParentMapID = -1

// enterMap déplace le joueur local vers la map de profondeur `depth`.
// Descente : récupère ou crée la map enfant (une seule par (parent, depth)).
// Remontée : retourne à la map parente (persistée dans le registre).
// Les autres maps (et les joueurs dessus) ne sont PAS affectées (fix bug 17/10 17:07).
func (s *State) enterMap(depth int) {
	cur := s.curMap()
	var m *Map
	ascending := cur != nil && depth < cur.Depth
	if ascending {
		// Remontée : la map parente persiste dans le registre.
		m = s.Maps.Get(cur.ParentMapID)
	} else if cur != nil {
		// Descente : une seule map par (parent, depth).
		m = s.Maps.MapAt(cur.ID, depth)
	}
	if m == nil {
		parentID := NoParentMapID
		if cur != nil {
			parentID = cur.ID
		}
		m = s.Maps.Create(MapKindDungeon, depth, parentID, 0, 0)
		s.populateDungeonMap(m)
	}
	s.enterMapInstance(m, ascending)
}

// enterMapInstance déplace le joueur local vers la map donnée (sans la créer).
// ascending=true : placement sur l'escalier montant si la map en a un.
func (s *State) enterMapInstance(m *Map, ascending bool) {
	// Place le joueur : sur l'escalier montant si on remonte, à l'entrée ou
	// sur une case libre sinon.
	px, py := m.EntranceX, m.EntranceY
	if ascending && m.World.HasStairsUp {
		px, py = m.World.StairsUpX, m.World.StairsUpY
	}
	if (px == 0 && py == 0) || !m.World.IsWalkable(px, py) || s.playerAtOnMap(m, px, py) != nil {
		px, py, _ = m.World.RandomFloor(NewRNG(s.Seed*1000 + int64(m.ID)))
	}
	s.Player.X, s.Player.Y = px, py
	s.Player.MapID = m.ID
	if m.FOVFor(s.Player.ID) == nil {
		m.SetFOV(s.Player.ID, NewFOV(m.World.Width*m.World.Height))
	}
	ComputeFOV(m.World, m.FOVFor(s.Player.ID), px, py, PlayerViewRange)
	if m.Depth >= 1 {
		if ascending {
			s.Logf("🌀 Étage %d/%d — bonne remontée !", m.Depth, MaxDepth)
		} else {
			s.Logf("🌀 Étage %d/%d — bonne descente !", m.Depth, MaxDepth)
		}
		if m.Depth == 1 {
			s.Logf("Le Seigneur des Tasses a volé tout le café. Trouve-le au étage %d !", MaxDepth)
		}
	}
}

// playerAtOnMap retourne le joueur à (x,y) sur la map donnée, ou nil.
func (s *State) playerAtOnMap(m *Map, x, y int) *Player {
	for _, p := range s.AllPlayers() {
		if p.MapID == m.ID && p.X == x && p.Y == y {
			return p
		}
	}
	return nil
}

// populateDungeonMap génère le contenu d'une map de donjon (world + spawns).
// Le RNG est dérivé (seed + mapID) pour des maps déterministes et stables.
func (s *State) populateDungeonMap(m *Map) {
	floorRNG := NewRNG(s.Seed*1000 + int64(m.ID))
	m.World = GenerateWorld(floorRNG, m.Depth, s.worldWidth, s.worldHeight)
	s.spawnEnemiesOn(m, floorRNG)
	s.spawnItemsOn(m, floorRNG)
	s.spawnMerchantOn(m, floorRNG)
	if m.Depth == MaxDepth {
		s.spawnBossOn(m, floorRNG)
	}
}

func (s *State) spawnEnemiesOn(m *Map, rng *RNG) {
	m.Entities.Reset()
	kinds := EnemyKindsForDepth(m.Depth)
	count := 3 + m.Depth
	if count > 12 {
		count = 12
	}
	for i := 0; i < count; i++ {
		x, y, ok := m.World.RandomFloor(rng)
		if !ok || s.occupiedOn(m, x, y) {
			continue
		}
		t := enemyTemplate(kinds[rng.Intn(len(kinds))])
		t.X, t.Y = x, y
		m.Entities.Spawn(t)
	}
}

// spawnBossOn invoque le boss final sur la map du dernier étage.
func (s *State) spawnBossOn(m *Map, rng *RNG) {
	x, y, ok := m.World.RandomFloor(rng)
	if !ok {
		return
	}
	b := enemyTemplate(KindBoss)
	b.X, b.Y = x, y
	m.Entities.Spawn(b)
	s.Logf("👑 Le Seigneur des Tasses t'attend... avec TON café !")
}

func (s *State) spawnItemsOn(m *Map, rng *RNG) {
	m.Items = nil
	count := RollItemCount(rng, m.Depth)
	for i := 0; i < count; i++ {
		x, y, ok := m.World.RandomFloor(rng)
		if !ok || s.occupiedOn(m, x, y) || itemAtOn(m, x, y) != nil {
			continue
		}
		m.Items = append(m.Items, RollItem(rng, x, y, m.Depth))
	}
}

// spawnMerchantOn invoque un PNJ marchand sur les maps de profondeur paire.
func (s *State) spawnMerchantOn(m *Map, rng *RNG) {
	m.Merchant = nil
	if m.Depth%2 != 0 || m.Depth >= MaxDepth || m.Kind != MapKindDungeon {
		return
	}
	for tries := 0; tries < 50; tries++ {
		x, y, ok := m.World.RandomFloor(rng)
		if !ok || s.occupiedOn(m, x, y) {
			continue
		}
		if (x == m.World.StairsX && y == m.World.StairsY) || (x == m.World.StairsUpX && y == m.World.StairsUpY) {
			continue
		}
		m.Merchant = NewMerchant(rng, x, y, m.Depth)
		s.Logf("💰 Un marchand campe par ici (touche m près de lui pour commercer).")
		return
	}
}

// occupiedOn indique si une case est bloquée sur la map donnée (joueur ou monstre).
func (s *State) occupiedOn(m *Map, x, y int) bool {
	if s.playerAtOnMap(m, x, y) != nil {
		return true
	}
	return m.Entities.EnemyAt(x, y) != InvalidEntity
}

// itemAtOn renvoie l'objet posé en (x, y) sur la map donnée, ou nil.
func itemAtOn(m *Map, x, y int) *Item {
	for _, it := range m.Items {
		if it.X == x && it.Y == y {
			return it
		}
	}
	return nil
}

// OccupiedByEnemy renvoie l'ID du monstre vivant en (x, y) sur la map courante.
func (s *State) OccupiedByEnemy(x, y int) EntityID {
	m := s.curMap()
	if m == nil {
		return InvalidEntity
	}
	return m.Entities.EnemyAt(x, y)
}

// Occupied indique si une case est bloquée sur la map courante.
func (s *State) Occupied(x, y int) bool {
	return s.occupiedOn(s.curMap(), x, y)
}

// ItemAt renvoie l'objet posé en (x, y) sur la map courante, ou nil.
func (s *State) ItemAt(x, y int) *Item {
	m := s.curMap()
	if m == nil {
		return nil
	}
	return itemAtOn(m, x, y)
}

func (s *State) removeItem(it *Item) {
	m := s.curMap()
	if m == nil {
		return
	}
	for i, o := range m.Items {
		if o == it {
			m.Items = append(m.Items[:i], m.Items[i+1:]...)
			return
		}
	}
}

// CanEnemyMoveTo indique si un monstre peut occuper (x, y) sur la map courante.
func (s *State) CanEnemyMoveTo(x, y int) bool {
	m := s.curMap()
	if m == nil || !m.World.IsWalkable(x, y) {
		return false
	}
	return s.playerAt(x, y) == nil && m.Entities.EnemyAt(x, y) == InvalidEntity
}

// MovePlayer déplace le joueur de (dx, dy) ou attaque un monstre adjacent.
// Retourne true si un tour a été consommé (mur = pas de tour consommé).
func (s *State) MovePlayer(dx, dy int) bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil {
		return false
	}
	nx, ny := s.Player.X+dx, s.Player.Y+dy
	if !m.World.InBounds(nx, ny) {
		return false
	}
	if id := m.Entities.EnemyAt(nx, ny); id != InvalidEntity {
		s.playerAttack(id)
		s.endPlayerTurn()
		return true
	}
	if !m.World.IsWalkable(nx, ny) {
		return false
	}
	s.Player.X, s.Player.Y = nx, ny
	s.pickupItems()
	s.endPlayerTurn()
	return true
}

// PlayerWait fait attendre le joueur un tour.
func (s *State) PlayerWait() {
	if s.Phase != PhasePlaying {
		return
	}
	s.endPlayerTurn()
}

// playerAttack résout l'attaque du joueur sur un monstre (arme incluse).
func (s *State) playerAttack(id EntityID) {
	m := s.curMap()
	if m == nil {
		return
	}
	e := m.Entities
	i := int(id)
	atk := s.Player.EffectiveAttack()
	dmg, crit, killed := Attack(s.RNG, atk, e.Defense[i], &e.HP[i])
	if crit {
		s.Logf("💥 Coup critique sur le %s ! -%d PV", e.Name[i], dmg)
	} else {
		s.Logf("Tu frappes le %s : -%d PV", e.Name[i], dmg)
	}
	if killed {
		xp := s.rollXP(e.XPValue[i])
		s.Logf("☠️ Le %s s'effondre ! (+%d XP)", e.Name[i], xp)
		s.Player.Kills++
		s.Player.GainXP(s, xp)
		if e.Kind[i] == KindBoss {
			s.victory()
		}
		e.Despawn(id)
	}
}

// rollXP applique une variance de ±20% à l'XP de base d'un monstre tué.
func (s *State) rollXP(base int) int {
	if base <= 0 {
		return 0
	}
	variance := 1 + (s.RNG.Float()*0.4 - 0.2)
	xp := int(float64(base) * variance)
	if xp < 1 {
		xp = 1
	}
	return xp
}

// endPlayerTurn termine le tour du joueur : régénération, FOV, puis tour des monstres.
func (s *State) endPlayerTurn() {
	s.Player.Turns++
	s.Turn = s.Player.Turns
	if s.Player.Turns%ManaRegenEvery == 0 && s.Player.Mana < s.Player.MaxMana {
		s.Player.Mana++
	}
	m := s.curMap()
	if m != nil {
		m.floorTurns++
		if fov := m.FOVFor(s.Player.ID); fov != nil {
			ComputeFOV(m.World, fov, s.Player.X, s.Player.Y, PlayerViewRange)
		}
	}
	s.enemiesTurn()
	s.checkDeath()
}

func (s *State) enemiesTurn() {
	m := s.curMap()
	if m == nil {
		return
	}
	e := m.Entities
	ids := make([]EntityID, 0, e.Len())
	e.EachAlive(func(id EntityID) { ids = append(ids, id) })
	for _, id := range ids {
		if !e.IsAlive(id) {
			continue
		}
		s.EnemyTurn(id)
		if s.Phase != PhasePlaying {
			return
		}
	}
	e.Compact()
}

func (s *State) checkDeath() {
	if s.Player.IsDead() && s.Phase == PhasePlaying {
		s.PlayerDeath()
	}
}

// PlayerDeath enregistre la défaite de la partie.
func (s *State) PlayerDeath() {
	if s.Phase != PhasePlaying {
		return
	}
	s.Phase = PhaseGameOver
	if s.CauseOfDeath == "" {
		s.CauseOfDeath = "mort dans le donjon"
	}
	s.Logf("💀 Tu es mort... le café reste au Seigneur des Tasses.")
}

// victory enregistre la victoire de la partie.
func (s *State) victory() {
	if s.Phase != PhasePlaying {
		return
	}
	s.Phase = PhaseVictory
	s.Won = true
	s.Logf("🏆 VICTOIRE ! Tu as vaincu le Seigneur des Tasses et récupéré le café !")
}

// Descend fait descendre le joueur d'un étage s'il se tient sur l'escalier.
// Change de map : la map parente persiste, les autres joueurs ne sont pas affectés.
func (s *State) Descend() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil || !m.World.HasStairs {
		s.Logf("Pas d'escalier ici.")
		return false
	}
	if s.Player.X != m.World.StairsX || s.Player.Y != m.World.StairsY {
		s.Logf("L'escalier est ailleurs sur cet étage.")
		return false
	}
	s.enterMap(m.Depth + 1)
	return true
}

// Ascend fait remonter le joueur d'un étage s'il se tient sur l'escalier montant.
func (s *State) Ascend() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	m := s.curMap()
	if m == nil || !m.World.HasStairsUp {
		s.Logf("Pas d'escalier montant ici (tu es au premier étage).")
		return false
	}
	if s.Player.X != m.World.StairsUpX || s.Player.Y != m.World.StairsUpY {
		s.Logf("L'escalier montant est ailleurs sur cet étage.")
		return false
	}
	s.enterMap(m.Depth - 1)
	return true
}

// pickupItems applique l'objet sous le joueur (sur la map courante).
func (s *State) pickupItems() {
	it := s.ItemAt(s.Player.X, s.Player.Y)
	if it == nil {
		return
	}
	switch it.Kind {
	case ItemGold:
		s.Player.Gold += it.Power
		s.Logf("Tu ramasses %d pièces d'or.", it.Power)
	case ItemWeapon, ItemArmor, ItemPotionHP, ItemPotionMana:
		s.Player.Inventory = append(s.Player.Inventory, it)
		s.Logf("🎒 %s ajouté à l'inventaire (i pour l'utiliser).", it.Name)
	}
	s.removeItem(it)
}

// EquipItem équipe l'objet à l'index donné (arme ou armure).
func (s *State) EquipItem(index int) bool {
	if index < 0 || index >= len(s.Player.Inventory) {
		return false
	}
	it := s.Player.Inventory[index]
	switch it.Kind {
	case ItemWeapon:
		if s.Player.WeaponName != "" {
			old := &Item{Kind: ItemWeapon, Name: s.Player.WeaponName, Glyph: '/', Power: s.Player.WeaponBonus}
			s.Player.Inventory = append(s.Player.Inventory, old)
		}
		s.Player.WeaponName = it.Name
		s.Player.WeaponBonus = it.Power
		s.Logf("⚔️ Tu équipes : %s (+%d attaque)", it.Name, it.Power)
	case ItemArmor:
		if s.Player.ArmorName != "" {
			old := &Item{Kind: ItemArmor, Name: s.Player.ArmorName, Glyph: '[', Power: s.Player.ArmorBonus}
			s.Player.Inventory = append(s.Player.Inventory, old)
		}
		s.Player.ArmorName = it.Name
		s.Player.ArmorBonus = it.Power
		s.Logf("🛡️ Tu équipes : %s (+%d défense)", it.Name, it.Power)
	default:
		return false
	}
	s.Player.Inventory = append(s.Player.Inventory[:index], s.Player.Inventory[index+1:]...)
	return true
}

// UseItem utilise l'objet à l'index donné (potion de soin ou de mana).
func (s *State) UseItem(index int) bool {
	if index < 0 || index >= len(s.Player.Inventory) {
		return false
	}
	it := s.Player.Inventory[index]
	switch it.Kind {
	case ItemPotionHP:
		s.Player.HP += it.Power
		if s.Player.HP > s.Player.MaxHP {
			s.Player.HP = s.Player.MaxHP
		}
		s.Logf("🧪 Potion de soin : +%d PV (%d/%d)", it.Power, s.Player.HP, s.Player.MaxHP)
	case ItemPotionMana:
		s.Player.Mana += it.Power
		if s.Player.Mana > s.Player.MaxMana {
			s.Player.Mana = s.Player.MaxMana
		}
		s.Logf("🧪 Potion de mana : +%d mana (%d/%d)", it.Power, s.Player.Mana, s.Player.MaxMana)
	default:
		return false
	}
	s.Player.Inventory = append(s.Player.Inventory[:index], s.Player.Inventory[index+1:]...)
	return true
}

// DropItem jette l'objet à l'index donné (le pose au sol sur la map courante).
func (s *State) DropItem(index int) {
	if index < 0 || index >= len(s.Player.Inventory) {
		return
	}
	it := s.Player.Inventory[index]
	it.X, it.Y = s.Player.X, s.Player.Y
	if m := s.curMap(); m != nil {
		m.Items = append(m.Items, it)
	}
	s.Player.Inventory = append(s.Player.Inventory[:index], s.Player.Inventory[index+1:]...)
	s.Logf("Tu jettes %s.", it.Name)
}

// InventoryValue renvoie la valeur totale de l'inventaire (pour la vente).
func (s *State) InventoryValue() int {
	total := 0
	for _, it := range s.Player.Inventory {
		total += ItemValue(it)
	}
	return total
}

// CastFireball lance une boule de feu sur la map courante.
func (s *State) CastFireball(lastDX, lastDY int) bool {
	if s.Phase != PhasePlaying {
		return false
	}
	if s.Player.Mana < FireballCost {
		s.Logf("Pas assez de mana (%d/%d).", s.Player.Mana, FireballCost)
		return false
	}
	m := s.curMap()
	if m == nil {
		return false
	}
	tx, ty := s.fireballTarget(lastDX, lastDY)
	s.Player.Mana -= FireballCost
	dmg := FireballBase + 2*s.Player.Level
	s.Logf("🔥 Boule de feu ! (%d dégâts en zone)", dmg)
	hit := false
	e := m.Entities
	ids := make([]EntityID, 0, e.Len())
	e.EachAlive(func(id EntityID) { ids = append(ids, id) })
	for _, id := range ids {
		i := int(id)
		if chebDist(e.X[i], e.Y[i], tx, ty) > 1 {
			continue
		}
		e.HP[i] -= dmg
		if e.HP[i] < 0 {
			e.HP[i] = 0
		}
		hit = true
		s.Logf("La boule de feu frappe le %s : -%d PV", e.Name[i], dmg)
		if e.HP[i] <= 0 {
			xp := s.rollXP(e.XPValue[i])
			s.Logf("☠️ Le %s s'effondre ! (+%d XP)", e.Name[i], xp)
			s.Player.Kills++
			s.Player.GainXP(s, xp)
			if e.Kind[i] == KindBoss {
				s.victory()
			}
			e.Despawn(id)
		}
	}
	if !hit {
		s.Logf("La boule de feu explose dans le vide.")
	}
	if s.Phase == PhasePlaying {
		s.endPlayerTurn()
	}
	return true
}

func (s *State) fireballTarget(lastDX, lastDY int) (int, int) {
	if tx, ty, ok := s.nearestVisibleEnemy(); ok {
		return tx, ty
	}
	return s.Player.X + lastDX, s.Player.Y + lastDY
}

// nearestVisibleEnemy renvoie le monstre vivant le plus proche visible sur la map courante.
func (s *State) nearestVisibleEnemy() (int, int, bool) {
	m := s.curMap()
	fov := s.curFOV()
	if m == nil || fov == nil {
		return 0, 0, false
	}
	bestDist := 1 << 30
	found := false
	bx, by := 0, 0
	e := m.Entities
	e.EachAlive(func(id EntityID) {
		i := int(id)
		if !fov.Visible[e.Y[i]*m.World.Width+e.X[i]] {
			return
		}
		if !s.hasLineOfSight(s.Player.X, s.Player.Y, e.X[i], e.Y[i]) {
			return
		}
		if d := chebDist(s.Player.X, s.Player.Y, e.X[i], e.Y[i]); d < bestDist {
			bestDist = d
			bx, by = e.X[i], e.Y[i]
			found = true
		}
	})
	return bx, by, found
}

// hasLineOfSight indique si la ligne entre deux cases ne traverse aucun mur (map courante).
func (s *State) hasLineOfSight(x0, y0, x1, y1 int) bool {
	m := s.curMap()
	if m == nil {
		return false
	}
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
		if x == x1 && y == y1 {
			return true
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
		if x == x1 && y == y1 {
			return true
		}
		if m.World.At(x, y) == TileWall {
			return false
		}
	}
}

// boss renvoie l'ID du boss vivant sur la map courante, ou InvalidEntity.
func (s *State) boss() EntityID {
	m := s.curMap()
	if m == nil {
		return InvalidEntity
	}
	return m.Entities.BossID()
}

// Score calcule le score de la partie.
func (s *State) Score() int {
	depth := 0
	if m := s.curMap(); m != nil {
		depth = m.Depth
	}
	score := s.Player.Gold*2 + s.Player.Kills*10 + s.Player.Level*25 + depth*50
	if s.Won {
		score += 1000
	}
	return score
}

// Logf ajoute un message au journal de la partie.
func (s *State) Logf(format string, args ...any) {
	s.Log = append(s.Log, fmt.Sprintf(format, args...))
	if len(s.Log) > MaxLog {
		s.Log = s.Log[len(s.Log)-MaxLog:]
	}
}

// Summary retourne un résumé textuel de la partie (fin ou en cours).
func (s *State) Summary() string {
	var result string
	switch s.Phase {
	case PhaseVictory:
		result = "VICTOIRE"
	case PhaseGameOver:
		result = "GAME OVER (" + s.CauseOfDeath + ")"
	default:
		result = "INACHEVÉ (limite de tours atteinte)"
	}
	depth := 0
	if m := s.curMap(); m != nil {
		depth = m.Depth
	}
	return fmt.Sprintf("[%s] seed=%d étage=%d niveau=%d PV=%d/%d or=%d kills=%d tours=%d score=%d",
		result, s.Seed, depth, s.Player.Level, s.Player.HP, s.Player.MaxHP,
		s.Player.Gold, s.Player.Kills, s.Player.Turns, s.Score())
}
