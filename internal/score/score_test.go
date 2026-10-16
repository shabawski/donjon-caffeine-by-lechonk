package score

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddLoadTop(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := Add(Entry{Seed: 1, Score: 100, Depth: 3}); err != nil {
		t.Fatal(err)
	}
	if err := Add(Entry{Seed: 2, Score: 300, Depth: 5, Won: true}); err != nil {
		t.Fatal(err)
	}
	if err := Add(Entry{Seed: 3, Score: 200, Depth: 4}); err != nil {
		t.Fatal(err)
	}

	entries, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("3 entrées attendues, got %d", len(entries))
	}
	if entries[0].Score != 300 || entries[1].Score != 200 || entries[2].Score != 100 {
		t.Fatalf("tri décroissant attendu, got %d/%d/%d", entries[0].Score, entries[1].Score, entries[2].Score)
	}
	if entries[0].Date == "" {
		t.Fatal("la date doit être renseignée par Add")
	}
	if !entries[0].Won {
		t.Fatal("le flag Won doit être persisté")
	}

	top := Top(2)
	if len(top) != 2 || top[0].Score != 300 || top[1].Score != 200 {
		t.Fatalf("Top(2) incorrect: %+v", top)
	}

	// Persistance sur disque.
	if _, err := os.Stat(filepath.Join(dir, "cafedungeon", "scores.json")); err != nil {
		t.Fatalf("fichier de scores absent: %v", err)
	}
}

func TestAddKeepsTop10(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for i := 0; i < 15; i++ {
		if err := Add(Entry{Seed: int64(i), Score: i * 10}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != maxEntries {
		t.Fatalf("top %d attendu, got %d", maxEntries, len(entries))
	}
	if entries[0].Score != 140 {
		t.Fatalf("meilleur score attendu 140, got %d", entries[0].Score)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	entries, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("liste vide attendue, got %d", len(entries))
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "cafedungeon"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cafedungeon", "scores.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("fichier corrompu => liste vide, pas d'erreur")
	}
}
