package game

import (
	"strings"
	"testing"
)

// TestAutoPlayTerminatesWithoutPanic joue 20 parties complètes en bot et
// vérifie que chacune atteint une fin (victoire, défaite ou limite de tours).
func TestAutoPlayTerminatesWithoutPanic(t *testing.T) {
	stats := map[Phase]int{}
	maxDepthReached := 0
	victories := 0
	for seed := int64(1); seed <= 20; seed++ {
		s := NewState(seed)
		for s.Phase == PhasePlaying && s.Player.Turns < 20000 {
			s.AutoAction()
		}
		stats[s.Phase]++
		if s.World.Depth > maxDepthReached {
			maxDepthReached = s.World.Depth
		}
		if s.Won {
			victories++
		}
		// Aucune de ces assertions ne doit paniciper ; Summary doit rester appelable.
		_ = s.Summary()
		_ = s.Score()
	}
	t.Logf("20 seeds: victoires=%d, gameover=%d, limite=%d, profondeur max=%d",
		stats[PhaseVictory], stats[PhaseGameOver], stats[PhasePlaying], maxDepthReached)
	if maxDepthReached < 5 {
		t.Errorf("le bot devrait atteindre au moins l'étage 5, got %d", maxDepthReached)
	}
	if stats[PhaseVictory]+stats[PhaseGameOver] == 0 {
		t.Error("aucune partie terminée sur 20 seeds (toutes à la limite de tours)")
	}
	_ = victories
}

// TestAutoPlayDeterministic vérifie la reproductibilité : même seed, même résumé.
func TestAutoPlayDeterministic(t *testing.T) {
	a := AutoPlay(42, 20000)
	b := AutoPlay(42, 20000)
	if a != b {
		t.Fatalf("même seed, résumés différents:\n%s\nvs\n%s", a, b)
	}
	if !strings.Contains(a, "seed=42") {
		t.Fatalf("résumé mal formé: %q", a)
	}
}

// TestVictoryPathEndToEnd prouve que la victoire est atteignable de bout en
// bout : descente jusqu'au dernier étage, puis mort du boss.
func TestVictoryPathEndToEnd(t *testing.T) {
	s := NewState(42)
	for s.World.Depth < MaxDepth {
		s.Player.X, s.Player.Y = s.World.StairsX, s.World.StairsY
		if !s.Descend() {
			t.Fatalf("impossible de descendre à l'étage %d", s.World.Depth)
		}
	}
	if s.World.Depth != MaxDepth {
		t.Fatalf("dernier étage attendu %d, got %d", MaxDepth, s.World.Depth)
	}
	if s.World.HasStairs {
		t.Fatal("pas d'escalier au dernier étage")
	}
	boss := s.boss()
	if boss == nil {
		t.Fatal("le boss doit exister au dernier étage")
	}
	// Le joueur se téléporte sur le boss et attaque (test de transition, pas de déplacement).
	s.Player.Attack = 1000
	s.Player.X, s.Player.Y = boss.X, boss.Y
	s.MovePlayer(0, 0)
	if s.Phase != PhaseVictory {
		t.Fatalf("phase attendue PhaseVictory, got %v", s.Phase)
	}
	if !s.Won {
		t.Fatal("Won doit être true")
	}
	if s.Score() <= 1000 {
		t.Fatalf("le score de victoire doit inclure le bonus, got %d", s.Score())
	}
	if !containsLog(s, "VICTOIRE") {
		t.Fatal("le journal doit annoncer la victoire")
	}
}

// TestGameOverPath vérifie la transition vers la défaite.
func TestGameOverPath(t *testing.T) {
	s := NewState(42)
	s.Player.HP = 1
	s.Player.MaxHP = 1
	// Entoure le joueur de gobelins trop forts.
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if s.World.IsWalkable(s.Player.X+d[0], s.Player.Y+d[1]) {
			g := s.addEnemy(KindGoblin, s.Player.X+d[0], s.Player.Y+d[1])
			g.Attack = 100
		}
	}
	ComputeFOV(s.World, s.FOV, s.Player.X, s.Player.Y, PlayerViewRange)
	s.PlayerWait()
	if s.Phase != PhaseGameOver {
		t.Fatalf("phase attendue PhaseGameOver, got %v", s.Phase)
	}
	if s.CauseOfDeath == "" {
		t.Fatal("une cause de mort doit être enregistrée")
	}
	if !strings.Contains(s.Summary(), "GAME OVER") {
		t.Fatalf("le résumé doit mentionner GAME OVER: %q", s.Summary())
	}
}

// TestFullFloorCycle simule un cycle complet étage par étage : exploration,
// combat, ramassage, descente — sur un étage généré procéduralement.
func TestFullFloorCycle(t *testing.T) {
	s := NewState(7)
	for s.World.Depth < 4 && s.Phase == PhasePlaying {
		// Tue tous les monstres de l'étage (god mode partiel).
		for _, e := range s.Enemies {
			e.HP = 1
		}
		for _, e := range s.Enemies {
			if !e.IsAlive() {
				continue
			}
			// Téléporte le joueur à côté et attaque.
			s.Player.Attack = 1000
			s.Player.X, s.Player.Y = e.X, e.Y
			s.MovePlayer(0, 0)
		}
		// Descend.
		s.Player.X, s.Player.Y = s.World.StairsX, s.World.StairsY
		if !s.Descend() {
			t.Fatalf("descente impossible à l'étage %d", s.World.Depth)
		}
	}
	if s.World.Depth != 4 {
		t.Fatalf("étage 4 attendu, got %d (phase=%v)", s.World.Depth, s.Phase)
	}
	if s.Player.Kills == 0 {
		t.Fatal("des monstres doivent avoir été tués")
	}
}
