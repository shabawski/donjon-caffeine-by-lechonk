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

// Player est le personnage contrôlé par le joueur.
// Reste une struct à part entière (pas une entité ECS) : le multijoueur
// pourra le promouvoir en entité plus tard si besoin.
type Player struct {
	Entity
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
}

// State est l'état complet d'une partie.
// Les monstres vivent dans Entities (registre ECS/SOA) ; le joueur reste
// une struct séparée ; les objets au sol restent un slice (peu nombreux).
type State struct {
	RNG          *RNG
	Seed         int64
	Phase        Phase
	World        *World
	FOV          *FOV
	Player       Player
	Entities     *Entities // registre ECS/SOA des monstres (remplace Enemies []*Entity)
	Items        []*Item
	Log          []string
	Turn         int
	Won          bool
	CauseOfDeath string
	worldWidth   int // dimensions d'étage, conservées entre les étages
	worldHeight  int
}

// NewState crée une nouvelle partie avec la graine donnée et génère l'étage 1
// aux dimensions par défaut (DefaultWidth x DefaultHeight).
func NewState(seed int64) *State {
	return NewStateSized(seed, DefaultWidth, DefaultHeight)
}

// NewStateSized crée une nouvelle partie avec des dimensions d'étage
// personnalisées (bornées par ClampWorldSize). Les dimensions sont conservées
// pour tous les étages de la partie.
func NewStateSized(seed int64, width, height int) *State {
	width, height = ClampWorldSize(width, height)
	s := &State{
		RNG:         NewRNG(seed),
		Seed:        seed,
		Phase:       PhasePlaying,
		Entities:    NewEntities(),
		worldWidth:  width,
		worldHeight: height,
	}
	s.Player = Player{
		Entity:  *NewPlayer(0, 0),
		Level:   1,
		MaxMana: StartMaxMana,
		Mana:    StartMaxMana,
	}
	s.startFloor(1)
	return s
}

// startFloor génère un étage et y place le joueur, les monstres et les objets.
func (s *State) startFloor(depth int) {
	s.World = GenerateWorld(s.RNG, depth, s.worldWidth, s.worldHeight)
	s.FOV = NewFOV(s.World.Width * s.World.Height)
	px, py, _ := s.World.RandomFloor(s.RNG)
	s.Player.X, s.Player.Y = px, py
	s.spawnEnemies(depth)
	s.spawnItems(depth)
	if depth == MaxDepth {
		s.spawnBoss()
	}
	ComputeFOV(s.World, s.FOV, px, py, PlayerViewRange)
	s.Logf("🌀 Étage %d/%d — bonne descente !", depth, MaxDepth)
	if depth == 1 {
		s.Logf("Le Seigneur des Tasses a volé tout le café. Trouve-le au étage %d !", MaxDepth)
	}
}

func (s *State) spawnEnemies(depth int) {
	s.Entities.Reset()
	kinds := EnemyKindsForDepth(depth)
	count := 3 + depth
	if count > 12 {
		count = 12
	}
	for i := 0; i < count; i++ {
		x, y, ok := s.World.RandomFloor(s.RNG)
		if !ok || s.Occupied(x, y) {
			continue
		}
		t := enemyTemplate(kinds[s.RNG.Intn(len(kinds))])
		t.X, t.Y = x, y
		s.Entities.Spawn(t)
	}
}

// spawnBoss invoque le boss final sur le dernier étage.
func (s *State) spawnBoss() {
	x, y, ok := s.World.RandomFloor(s.RNG)
	if !ok {
		return
	}
	b := enemyTemplate(KindBoss)
	b.X, b.Y = x, y
	s.Entities.Spawn(b)
	s.Logf("👑 Le Seigneur des Tasses t'attend... avec TON café !")
}

func (s *State) spawnItems(depth int) {
	s.Items = nil
	count := RollItemCount(s.RNG, depth)
	for i := 0; i < count; i++ {
		x, y, ok := s.World.RandomFloor(s.RNG)
		if !ok || s.Occupied(x, y) || s.ItemAt(x, y) != nil {
			continue
		}
		s.Items = append(s.Items, RollItem(s.RNG, x, y, depth))
	}
}

// OccupiedByEnemy renvoie l'ID du monstre vivant en (x, y), ou InvalidEntity.
func (s *State) OccupiedByEnemy(x, y int) EntityID {
	return s.Entities.EnemyAt(x, y)
}

// Occupied indique si une case est bloquée (joueur ou monstre vivant).
func (s *State) Occupied(x, y int) bool {
	if s.Player.X == x && s.Player.Y == y {
		return true
	}
	return s.OccupiedByEnemy(x, y) != InvalidEntity
}

// ItemAt renvoie l'objet posé en (x, y), ou nil.
func (s *State) ItemAt(x, y int) *Item {
	for _, it := range s.Items {
		if it.X == x && it.Y == y {
			return it
		}
	}
	return nil
}

func (s *State) removeItem(it *Item) {
	for i, o := range s.Items {
		if o == it {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			return
		}
	}
}

// CanEnemyMoveTo indique si un monstre peut occuper (x, y).
func (s *State) CanEnemyMoveTo(x, y int) bool {
	if !s.World.IsWalkable(x, y) {
		return false
	}
	if s.Player.X == x && s.Player.Y == y {
		return false
	}
	return s.OccupiedByEnemy(x, y) == InvalidEntity
}

// MovePlayer déplace le joueur de (dx, dy) ou attaque un monstre adjacent.
// Retourne true si un tour a été consommé (mur = pas de tour consommé).
func (s *State) MovePlayer(dx, dy int) bool {
	if s.Phase != PhasePlaying {
		return false
	}
	nx, ny := s.Player.X+dx, s.Player.Y+dy
	if !s.World.InBounds(nx, ny) {
		return false
	}
	if id := s.OccupiedByEnemy(nx, ny); id != InvalidEntity {
		s.playerAttack(id)
		s.endPlayerTurn()
		return true
	}
	if !s.World.IsWalkable(nx, ny) {
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
	e := s.Entities
	i := int(id)
	atk := s.Player.EffectiveAttack()
	dmg, crit, killed := Attack(s.RNG, atk, e.Defense[i], &e.HP[i])
	if crit {
		s.Logf("💥 Coup critique sur le %s ! -%d PV", e.Name[i], dmg)
	} else {
		s.Logf("Tu frappes le %s : -%d PV", e.Name[i], dmg)
	}
	if killed {
		s.Logf("☠️ Le %s s'effondre ! (+%d XP)", e.Name[i], e.XPValue[i])
		s.Player.Kills++
		s.Player.GainXP(s, e.XPValue[i])
		if e.Kind[i] == KindBoss {
			s.victory()
		}
		e.Despawn(id)
	}
}

// endPlayerTurn termine le tour du joueur : régénération, FOV, puis tour des monstres.
func (s *State) endPlayerTurn() {
	s.Player.Turns++
	s.Turn = s.Player.Turns
	if s.Player.Turns%ManaRegenEvery == 0 && s.Player.Mana < s.Player.MaxMana {
		s.Player.Mana++
	}
	ComputeFOV(s.World, s.FOV, s.Player.X, s.Player.Y, PlayerViewRange)
	s.enemiesTurn()
	s.checkDeath()
}

func (s *State) enemiesTurn() {
	e := s.Entities
	// Collecte les IDs vivants avant la boucle : Compact (fin de tour)
	// invalide les indices, et EnemyTurn peut tuer (Despawn) des monstres.
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
func (s *State) Descend() bool {
	if s.Phase != PhasePlaying {
		return false
	}
	if !s.World.HasStairs {
		s.Logf("Pas d'escalier ici.")
		return false
	}
	if s.Player.X != s.World.StairsX || s.Player.Y != s.World.StairsY {
		s.Logf("L'escalier est ailleurs sur cet étage.")
		return false
	}
	s.startFloor(s.World.Depth + 1)
	return true
}

// pickupItems applique l'objet sous le joueur (ramassage automatique).
func (s *State) pickupItems() {
	it := s.ItemAt(s.Player.X, s.Player.Y)
	if it == nil {
		return
	}
	switch it.Kind {
	case ItemGold:
		s.Player.Gold += it.Power
		s.Logf("Tu ramasses %d pièces d'or.", it.Power)
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
	case ItemWeapon:
		if it.Power > s.Player.WeaponBonus {
			s.Logf("⚔️ Tu équipes : %s (+%d attaque)", it.Name, it.Power)
			s.Player.WeaponName = it.Name
			s.Player.WeaponBonus = it.Power
		} else {
			s.Logf("Tu laisses %s (ton arme est meilleure).", it.Name)
		}
	case ItemArmor:
		if it.Power > s.Player.ArmorBonus {
			s.Logf("🛡️ Tu équipes : %s (+%d défense)", it.Name, it.Power)
			s.Player.ArmorName = it.Name
			s.Player.ArmorBonus = it.Power
		} else {
			s.Logf("Tu laisses %s (ton armure est meilleure).", it.Name)
		}
	}
	s.removeItem(it)
}

// CastFireball lance une boule de feu : dégâts fixes (base + 2/niveau) sur tous
// les monstres dans les 8 cases autour de la cible. La cible est le monstre
// visible le plus proche (ligne de vue dégagée) ; à défaut la case devant le
// joueur selon sa dernière direction. Coûte FireballCost mana.
func (s *State) CastFireball(lastDX, lastDY int) bool {
	if s.Phase != PhasePlaying {
		return false
	}
	if s.Player.Mana < FireballCost {
		s.Logf("Pas assez de mana (%d/%d).", s.Player.Mana, FireballCost)
		return false
	}
	tx, ty := s.fireballTarget(lastDX, lastDY)
	s.Player.Mana -= FireballCost
	dmg := FireballBase + 2*s.Player.Level
	s.Logf("🔥 Boule de feu ! (%d dégâts en zone)", dmg)
	hit := false
	e := s.Entities
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
			s.Logf("☠️ Le %s s'effondre ! (+%d XP)", e.Name[i], e.XPValue[i])
			s.Player.Kills++
			s.Player.GainXP(s, e.XPValue[i])
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

// nearestVisibleEnemy renvoie le monstre vivant le plus proche actuellement
// visible par le joueur avec une ligne de vue dégagée.
func (s *State) nearestVisibleEnemy() (int, int, bool) {
	bestDist := 1 << 30
	found := false
	bx, by := 0, 0
	e := s.Entities
	e.EachAlive(func(id EntityID) {
		i := int(id)
		if !s.FOV.Visible[e.Y[i]*s.World.Width+e.X[i]] {
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

// hasLineOfSight indique si la ligne entre deux cases ne traverse aucun mur.
func (s *State) hasLineOfSight(x0, y0, x1, y1 int) bool {
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
		if s.World.At(x, y) == TileWall {
			return false
		}
	}
}

// boss renvoie l'ID du boss vivant de l'étage, ou InvalidEntity.
func (s *State) boss() EntityID {
	return s.Entities.BossID()
}

// Score calcule le score de la partie.
func (s *State) Score() int {
	score := s.Player.Gold*2 + s.Player.Kills*10 + s.Player.Level*25 + s.World.Depth*50
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
	return fmt.Sprintf("[%s] seed=%d étage=%d niveau=%d PV=%d/%d or=%d kills=%d tours=%d score=%d",
		result, s.Seed, s.World.Depth, s.Player.Level, s.Player.HP, s.Player.MaxHP,
		s.Player.Gold, s.Player.Kills, s.Player.Turns, s.Score())
}
