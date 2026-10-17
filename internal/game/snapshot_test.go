package game

import (
	"encoding/json"
	"testing"
)

// TestSnapshotCapturesState vérifie que le snapshot capture les données visibles.
func TestSnapshotCapturesState(t *testing.T) {
	s := NewState(42)
	snap := s.Snapshot()
	if snap.Seed != 42 {
		t.Fatalf("seed attendue 42, got %d", snap.Seed)
	}
	if snap.Depth != 1 {
		t.Fatalf("étage attendu 1, got %d", snap.Depth)
	}
	if snap.Width != s.World.Width || snap.Height != s.World.Height {
		t.Fatalf("dimensions %dx%d, attendu %dx%d", snap.Width, snap.Height, s.World.Width, s.World.Height)
	}
	if len(snap.Tiles) != len(s.World.Tiles) {
		t.Fatalf("tiles: %d, attendu %d", len(snap.Tiles), len(s.World.Tiles))
	}
	if snap.Player.X != s.Player.X || snap.Player.Y != s.Player.Y {
		t.Fatalf("position joueur (%d,%d), attendu (%d,%d)", snap.Player.X, snap.Player.Y, s.Player.X, s.Player.Y)
	}
	if snap.Player.Level != s.Player.Level || snap.Player.Gold != s.Player.Gold {
		t.Fatalf("stats joueur divergent: %+v", snap.Player)
	}
	if len(snap.Enemies) != s.Entities.Len() {
		t.Fatalf("ennemis %d, attendu %d", len(snap.Enemies), s.Entities.Len())
	}
	if len(snap.Visible) != len(s.FOV.Visible) || len(snap.Explored) != len(s.FOV.Explored) {
		t.Fatal("FOV mal dimensionné")
	}
	if len(snap.Log) != len(s.Log) {
		t.Fatalf("log %d lignes, attendu %d", len(snap.Log), len(s.Log))
	}
	if snap.Phase != s.Phase {
		t.Fatalf("phase %v, attendu %v", snap.Phase, s.Phase)
	}
}

// TestSnapshotHelpers vérifie les méthodes d'accès du snapshot (rendu TUI).
func TestSnapshotHelpers(t *testing.T) {
	s := NewState(42)
	snap := s.Snapshot()
	// InBounds / At
	if !snap.InBounds(0, 0) || snap.InBounds(-1, 0) || snap.InBounds(snap.Width, snap.Height) {
		t.Fatal("InBounds incorrect")
	}
	if snap.At(-1, 0) != TileWall {
		t.Fatal("hors limites doit valoir TileWall")
	}
	// IsWalkable sur la case du joueur
	if !snap.IsWalkable(snap.Player.X, snap.Player.Y) {
		t.Fatal("la case du joueur doit être franchissable")
	}
	// EnemyAt
	found := false
	for _, e := range snap.Enemies {
		if got := snap.EnemyAt(e.X, e.Y); got == nil || got.Kind != e.Kind {
			t.Fatalf("EnemyAt(%d,%d) doit retourner %v", e.X, e.Y, e.Kind)
		}
		found = true
	}
	if !found {
		t.Fatal("au moins un ennemi attendu à l'étage 1")
	}
	if snap.EnemyAt(0, 0) != nil {
		t.Fatal("pas d'ennemi attendu en (0,0) (mur de bordure)")
	}
	// ItemAt
	if snap.ItemAt(snap.Player.X, snap.Player.Y) != nil {
		// OK si pas d'item sous le joueur
	}
	// Score cohérent avec State.Score
	if snap.Score() != s.Score() {
		t.Fatalf("score snapshot %d != state %d", snap.Score(), s.Score())
	}
}

// TestSnapshotMerchant vérifie que le marchand est capturé sur les étages pairs.
func TestSnapshotMerchant(t *testing.T) {
	s := NewState(42)
	// Étage 1 (impair) : pas de marchand
	if snap := s.Snapshot(); snap.Merchant != nil {
		t.Fatal("pas de marchand attendu à l'étage 1")
	}
	// Descend à l'étage 2 (pair) : marchand attendu
	s.Player.X, s.Player.Y = s.World.StairsX, s.World.StairsY
	if !s.Descend() {
		t.Fatal("descente impossible")
	}
	snap := s.Snapshot()
	if snap.Merchant == nil {
		t.Fatal("marchand attendu à l'étage 2")
	}
	if snap.Merchant.Glyph != 'M' {
		t.Fatalf("glyph marchand 'M' attendu, got %c", snap.Merchant.Glyph)
	}
	if len(snap.Merchant.Shop) != 6 {
		t.Fatalf("boutique de 6 objets attendue, got %d", len(snap.Merchant.Shop))
	}
}

// TestSnapshotJSONRoundTrip vérifie que le snapshot survit à une
// sérialisation JSON (indispensable pour le transport TCP).
func TestSnapshotJSONRoundTrip(t *testing.T) {
	s := NewState(42)
	snap := s.Snapshot()
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Snapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Seed != snap.Seed || decoded.Depth != snap.Depth {
		t.Fatal("champs scalaires perdus après round-trip JSON")
	}
	if len(decoded.Tiles) != len(snap.Tiles) {
		t.Fatal("tuiles perdues après round-trip JSON")
	}
	if decoded.Player.X != snap.Player.X || decoded.Player.HP != snap.Player.HP {
		t.Fatal("joueur perdu après round-trip JSON")
	}
	if len(decoded.Enemies) != len(snap.Enemies) {
		t.Fatal("ennemis perdus après round-trip JSON")
	}
	if len(decoded.Visible) != len(snap.Visible) {
		t.Fatal("FOV perdu après round-trip JSON")
	}
	// Le score recalculé doit être identique.
	if decoded.Score() != snap.Score() {
		t.Fatal("score divergent après round-trip JSON")
	}
}

// TestSnapshotEffectiveStats vérifie que les stats effectives sont précalculées.
func TestSnapshotEffectiveStats(t *testing.T) {
	s := newTestState(1, 10, 10)
	s.Player.WeaponBonus = 3
	s.Player.ArmorBonus = 2
	snap := s.Snapshot()
	if snap.Player.EffectiveAtk != 8 || snap.Player.EffectiveDef != 3 {
		t.Fatalf("stats effectives attendues 8/3, got %d/%d",
			snap.Player.EffectiveAtk, snap.Player.EffectiveDef)
	}
}
