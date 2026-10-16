package game

import (
	"strings"
	"testing"
)

func TestNewState(t *testing.T) {
	s := NewState(123)
	if s.Phase != PhasePlaying {
		t.Fatalf("phase initiale %v, attendu PhasePlaying", s.Phase)
	}
	if s.World == nil || s.World.Depth != 1 {
		t.Fatal("étage 1 attendu")
	}
	if !s.World.HasStairs {
		t.Fatal("l'étage 1 doit avoir un escalier")
	}
	if !s.World.IsWalkable(s.Player.X, s.Player.Y) {
		t.Fatal("le joueur doit spawner sur une case franchissable")
	}
	if !s.FOV.Visible[s.Player.Y*s.World.Width+s.Player.X] {
		t.Fatal("le FOV doit être calculé au spawn")
	}
	if len(s.Enemies) < 1 {
		t.Fatal("des monstres doivent spawner à l'étage 1")
	}
	if len(s.Log) < 2 {
		t.Fatal("le journal doit accueillir le joueur")
	}
	if s.Player.Level != 1 || s.Player.Mana != StartMaxMana || s.Player.MaxMana != StartMaxMana {
		t.Fatal("stats de départ incorrectes")
	}
}

func TestMovePlayerIntoWall(t *testing.T) {
	s := newTestState(1, 10, 10) // joueur en (1,1), (0,1) est un mur de bordure
	if s.MovePlayer(-1, 0) {
		t.Fatal("un mur ne doit pas consommer de tour")
	}
	if s.Player.X != 1 || s.Player.Y != 1 {
		t.Fatal("le joueur ne doit pas traverser un mur")
	}
	if s.Player.Turns != 0 {
		t.Fatal("aucun tour ne doit être consommé")
	}
}

func TestMovePlayerOnFloor(t *testing.T) {
	s := newTestState(1, 10, 10)
	if !s.MovePlayer(1, 0) {
		t.Fatal("un déplacement sur le sol doit réussir")
	}
	if s.Player.X != 2 || s.Player.Y != 1 {
		t.Fatalf("position attendue (2,1), got (%d,%d)", s.Player.X, s.Player.Y)
	}
	if s.Player.Turns != 1 {
		t.Fatalf("un tour consommé, got %d", s.Player.Turns)
	}
}

func TestMovePlayerAttacksEnemy(t *testing.T) {
	s := newTestState(1, 10, 10)
	rat := s.addEnemy(KindRat, 2, 1)
	rat.HP = 1
	s.Player.Attack = 100
	if !s.MovePlayer(1, 0) {
		t.Fatal("attaquer un monstre consomme un tour")
	}
	if rat.IsAlive() {
		t.Fatal("le rat avec 1 PV doit mourir")
	}
	if s.Player.Kills != 1 {
		t.Fatalf("kills attendu 1, got %d", s.Player.Kills)
	}
	if s.Player.XP != rat.XPValue {
		t.Fatalf("XP attendu %d, got %d", rat.XPValue, s.Player.XP)
	}
	// Le monstre mort est retiré au tour suivant des monstres.
	if len(s.Enemies) != 0 {
		t.Fatalf("le monstre mort doit être retiré, %d restant(s)", len(s.Enemies))
	}
}

func TestMovePlayerBlockedWhenNotPlaying(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Phase = PhaseGameOver
	if s.MovePlayer(1, 0) {
		t.Fatal("aucun mouvement possible hors PhasePlaying")
	}
}

func TestPickupPotionHeals(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.HP = 10
	s.Items = []*Item{{Kind: ItemPotionHP, Name: "potion de soin", Glyph: '!', X: 2, Y: 1, Power: 10}}
	s.MovePlayer(1, 0)
	if s.Player.HP != 20 {
		t.Fatalf("PV attendu 20 après potion, got %d", s.Player.HP)
	}
	if len(s.Items) != 0 {
		t.Fatal("la potion doit être retirée de la carte")
	}
}

func TestPickupPotionClampsToMaxHP(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.HP = 28
	s.Items = []*Item{{Kind: ItemPotionHP, Name: "potion de soin", Glyph: '!', X: 2, Y: 1, Power: 10}}
	s.MovePlayer(1, 0)
	if s.Player.HP != s.Player.MaxHP {
		t.Fatalf("PV doit être plafonné à %d, got %d", s.Player.MaxHP, s.Player.HP)
	}
}

func TestPickupBetterWeaponEquips(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Items = []*Item{{Kind: ItemWeapon, Name: "épée courte", Glyph: '/', X: 2, Y: 1, Power: 3}}
	s.MovePlayer(1, 0)
	if s.Player.WeaponBonus != 3 || s.Player.WeaponName != "épée courte" {
		t.Fatalf("arme attendue équipée, got %q (+%d)", s.Player.WeaponName, s.Player.WeaponBonus)
	}
}

func TestPickupWorseWeaponIgnored(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.WeaponBonus = 5
	s.Player.WeaponName = "marteau de forge"
	s.Items = []*Item{{Kind: ItemWeapon, Name: "dague rouillée", Glyph: '/', X: 2, Y: 1, Power: 1}}
	s.MovePlayer(1, 0)
	if s.Player.WeaponBonus != 5 || s.Player.WeaponName != "marteau de forge" {
		t.Fatal("une arme plus faible ne doit pas être équipée")
	}
}

func TestPickupGold(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Items = []*Item{{Kind: ItemGold, Name: "pièces d'or", Glyph: '$', X: 2, Y: 1, Power: 42}}
	s.MovePlayer(1, 0)
	if s.Player.Gold != 42 {
		t.Fatalf("or attendu 42, got %d", s.Player.Gold)
	}
}

func TestCastFireballCostsManaAndDamagesArea(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.X, s.Player.Y = 7, 7
	s.Player.Mana = 20
	rat1 := s.addEnemy(KindRat, 8, 7)
	rat2 := s.addEnemy(KindRat, 7, 8)
	ComputeFOV(s.World, s.FOV, 7, 7, PlayerViewRange)
	if !s.CastFireball(0, 0) {
		t.Fatal("la boule de feu doit être lancée")
	}
	if s.Player.Mana != 20-FireballCost {
		t.Fatalf("mana attendu %d, got %d", 20-FireballCost, s.Player.Mana)
	}
	dmg := FireballBase + 2*s.Player.Level // 10
	if rat1.HP != 8-dmg || rat2.HP != 8-dmg {
		t.Fatalf("dégâts de zone attendus %d: rat1=%d rat2=%d", dmg, rat1.HP, rat2.HP)
	}
	if s.Player.Turns != 1 {
		t.Fatal("la boule de feu consomme un tour")
	}
}

func TestCastFireballKillsAndGrantsXP(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.X, s.Player.Y = 7, 7
	rat := s.addEnemy(KindRat, 8, 7) // 8 PV, fireball niveau 1 = 10 dégâts
	ComputeFOV(s.World, s.FOV, 7, 7, PlayerViewRange)
	s.CastFireball(0, 0)
	if rat.IsAlive() {
		t.Fatal("le rat doit mourir dans la boule de feu")
	}
	if s.Player.Kills != 1 {
		t.Fatalf("kills attendu 1, got %d", s.Player.Kills)
	}
}

func TestCastFireballNotEnoughMana(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.Mana = FireballCost - 1
	if s.CastFireball(0, 0) {
		t.Fatal("pas assez de mana: le sort doit échouer")
	}
	if s.Player.Mana != FireballCost-1 {
		t.Fatal("le mana ne doit pas être débité en cas d'échec")
	}
	if s.Player.Turns != 0 {
		t.Fatal("un sort échoué ne consomme pas de tour")
	}
}

func TestManaRegen(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.Mana = 0
	for i := 0; i < ManaRegenEvery; i++ {
		s.PlayerWait()
	}
	if s.Player.Mana != 1 {
		t.Fatalf("régénération +1 mana tous les %d tours, got %d", ManaRegenEvery, s.Player.Mana)
	}
}

func TestDescendRequiresStairs(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.World.HasStairs = true
	s.World.StairsX, s.World.StairsY = 5, 5
	s.World.Set(5, 5, TileStairs)
	if s.Descend() {
		t.Fatal("descendre hors de l'escalier doit échouer")
	}
	s.Player.X, s.Player.Y = 5, 5
	if !s.Descend() {
		t.Fatal("descendre sur l'escalier doit réussir")
	}
	if s.World.Depth != 2 {
		t.Fatalf("étage attendu 2, got %d", s.World.Depth)
	}
}

func TestDescendWithoutStairs(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.World.HasStairs = false
	if s.Descend() {
		t.Fatal("pas d'escalier: descente impossible")
	}
}

func TestVictoryOnBossKill(t *testing.T) {
	s := newTestState(1, 10, 10)
	boss := s.addEnemy(KindBoss, 2, 1)
	boss.HP = 1
	s.Player.Attack = 100
	s.MovePlayer(1, 0) // attaque le boss
	if s.Phase != PhaseVictory {
		t.Fatalf("phase attendue PhaseVictory, got %v", s.Phase)
	}
	if !s.Won {
		t.Fatal("Won doit être true")
	}
	if !containsLog(s, "VICTOIRE") {
		t.Fatal("le journal doit annoncer la victoire")
	}
}

func TestVictoryViaFireball(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.X, s.Player.Y = 7, 7
	s.Player.Level = 10 // fireball = 8 + 20 = 28 dégâts > 60 PV du boss? non, boss a 60 PV
	boss := s.addEnemy(KindBoss, 8, 7)
	boss.HP = 10
	ComputeFOV(s.World, s.FOV, 7, 7, PlayerViewRange)
	s.CastFireball(0, 0)
	if s.Phase != PhaseVictory {
		t.Fatalf("tuer le boss avec une boule de feu doit gagner, phase=%v", s.Phase)
	}
}

func TestPlayerDeath(t *testing.T) {
	s := newTestState(1, 15, 15)
	s.Player.X, s.Player.Y = 5, 5
	s.Player.HP = 1
	gob := s.addEnemy(KindGoblin, 6, 5)
	gob.Attack = 50
	ComputeFOV(s.World, s.FOV, 5, 5, PlayerViewRange)
	s.PlayerWait() // le gobelin attaque et tue le joueur
	if s.Phase != PhaseGameOver {
		t.Fatalf("phase attendue PhaseGameOver, got %v", s.Phase)
	}
	if !s.Player.IsDead() {
		t.Fatal("le joueur doit être mort")
	}
	if s.CauseOfDeath != "tué par un(e) gobelin" {
		t.Fatalf("cause de mort attendue, got %q", s.CauseOfDeath)
	}
	if !containsLog(s, "Tu es mort") {
		t.Fatal("le journal doit annoncer la mort")
	}
}

func TestNoActionsAfterGameOver(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Phase = PhaseGameOver
	s.PlayerWait()
	s.CastFireball(1, 0)
	s.Descend()
	if s.Player.Turns != 0 {
		t.Fatal("aucune action ne doit être possible après la fin")
	}
}

func TestScoreCalculation(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.Gold = 10
	s.Player.Kills = 5
	s.Player.Level = 3
	s.World.Depth = 2
	// 10*2 + 5*10 + 3*25 + 2*50 = 245
	if got := s.Score(); got != 245 {
		t.Fatalf("score attendu 245, got %d", got)
	}
	s.Won = true
	if got := s.Score(); got != 1245 {
		t.Fatalf("score avec victoire attendu 1245, got %d", got)
	}
}

func TestLogfCap(t *testing.T) {
	s := newTestState(1, 10, 10)
	for i := 0; i < MaxLog+50; i++ {
		s.Logf("message %d", i)
	}
	if len(s.Log) != MaxLog {
		t.Fatalf("le journal doit être plafonné à %d, got %d", MaxLog, len(s.Log))
	}
	if !strings.Contains(s.Log[len(s.Log)-1], "message") {
		t.Fatal("les derniers messages doivent être conservés")
	}
}

func TestSummary(t *testing.T) {
	s := NewState(42)
	sum := s.Summary()
	for _, want := range []string{"seed=42", "étage=", "score="} {
		if !strings.Contains(sum, want) {
			t.Fatalf("Summary() doit contenir %q, got %q", want, sum)
		}
	}
}

func TestOccupiedAndItemAt(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.X, s.Player.Y = 3, 3
	if !s.Occupied(3, 3) {
		t.Fatal("la case du joueur est occupée")
	}
	rat := s.addEnemy(KindRat, 5, 5)
	if !s.Occupied(5, 5) || s.OccupiedByEnemy(5, 5) != rat {
		t.Fatal("la case du monstre est occupée")
	}
	rat.HP = 0
	if s.Occupied(5, 5) {
		t.Fatal("un monstre mort n'occupe plus sa case")
	}
	if s.ItemAt(1, 1) != nil {
		t.Fatal("pas d'objet en (1,1)")
	}
	s.Items = []*Item{{Kind: ItemGold, X: 2, Y: 2}}
	if s.ItemAt(2, 2) == nil {
		t.Fatal("objet attendu en (2,2)")
	}
}

func containsLog(s *State, substr string) bool {
	for _, line := range s.Log {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}
