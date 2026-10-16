// Package score gère la persistance des meilleurs scores du Donjon Caféiné.
package score

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry est un score enregistré à la fin d'une partie.
type Entry struct {
	Date  string `json:"date"`
	Seed  int64  `json:"seed"`
	Depth int    `json:"depth"`
	Level int    `json:"level"`
	Gold  int    `json:"gold"`
	Kills int    `json:"kills"`
	Turns int    `json:"turns"`
	Score int    `json:"score"`
	Won   bool   `json:"won"`
	Death string `json:"death,omitempty"`
}

type file struct {
	Entries []Entry `json:"entries"`
}

const maxEntries = 10

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cafedungeon", "scores.json"), nil
}

// Load lit les scores enregistrés (liste vide si le fichier n'existe pas).
func Load() ([]Entry, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return []Entry{}, nil // fichier corrompu : on repart de zéro
	}
	return f.Entries, nil
}

// Add enregistre un score et persiste le top 10 (écriture atomique temp+rename).
func Add(e Entry) error {
	entries, err := Load()
	if err != nil {
		return err
	}
	e.Date = time.Now().Format("2006-01-02 15:04")
	entries = append(entries, e)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Score > entries[j].Score })
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(file{Entries: entries}, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Top retourne les n meilleurs scores (les plus hauts d'abord).
func Top(n int) []Entry {
	entries, _ := Load()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Score > entries[j].Score })
	if len(entries) > n {
		entries = entries[:n]
	}
	return entries
}
