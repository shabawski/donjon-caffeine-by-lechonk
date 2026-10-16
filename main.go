// Command donjon-cafeine est le point d'entrée du jeu « Le Donjon Caféiné ».
//
// Usage :
//
//	donjon-cafeine                  partie jouable (TUI terminal)
//	donjon-cafeine --seed 42        partie reproductible
//	donjon-cafeine --demo --seed 42 partie automatique headless (vérification)
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"game-test/internal/game"
	"game-test/internal/tui"
)

func main() {
	seed := flag.Int64("seed", 0, "graine de la partie (0 = aléatoire)")
	demo := flag.Bool("demo", false, "partie automatique headless (vérification de bout en bout)")
	maxTurns := flag.Int("max-turns", 30000, "limite de tours en mode --demo")
	flag.Parse()

	s := *seed
	if s == 0 {
		s = time.Now().UnixNano()
	}

	if *demo {
		fmt.Println(game.AutoPlay(s, *maxTurns))
		return
	}

	m := tui.New(game.NewState(s))
	if err := m.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
		os.Exit(1)
	}
}
