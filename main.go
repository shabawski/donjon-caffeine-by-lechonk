// Command donjon-cafeine est le point d'entrée du jeu « Le Donjon Caféiné ».
//
// Architecture client/serveur : le TUI ne touche plus au moteur directement.
// En solo (défaut), un serveur embarqué tourne dans le même process et
// communique par channels (client.LocalClient). En multijoueur (--connect),
// le client se connecte en TCP à un serveur distant (cmd/coffee-dungeon-server).
//
// Usage :
//
//	donjon-cafeine                    partie solo (serveur local via channels)
//	donjon-cafeine --seed 42          partie solo reproductible
//	donjon-cafeine --connect host:port  multijoueur (serveur TCP distant)
//	donjon-cafeine --demo --seed 42   partie automatique headless (vérification)
//	donjon-cafeine --demo --mode mmorpg --seed 42   démo MMORPG (monde ouvert)
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"game-test/internal/client"
	"game-test/internal/game"
	"game-test/internal/tui"
)

func main() {
	seed := flag.Int64("seed", 0, "graine de la partie (0 = aléatoire, solo uniquement)")
	demo := flag.Bool("demo", false, "partie automatique headless (vérification de bout en bout)")
	maxTurns := flag.Int("max-turns", 30000, "limite de tours en mode --demo")
	connect := flag.String("connect", "", "se connecter à un serveur distant (host:port, multijoueur)")
	mode := flag.String("mode", "coop", "mode de la partie --demo : coop (donjon) ou mmorpg (monde ouvert)")
	flag.Parse()

	if *demo {
		s := *seed
		if s == 0 {
			s = time.Now().UnixNano()
		}
		if *mode == "mmorpg" {
			fmt.Println(game.AutoPlayMMORPG(s, *maxTurns))
		} else {
			fmt.Println(game.AutoPlay(s, *maxTurns))
		}
		return
	}

	var gc client.GameClient
	if *connect != "" {
		// Multijoueur : connexion TCP à un serveur distant (bloc 5d/6).
		tcp, err := client.NewTCPClient(*connect)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Erreur connexion:", err)
			os.Exit(1)
		}
		gc = tcp
	} else {
		// Solo : serveur embarqué dans le même process (channels).
		s := *seed
		if s == 0 {
			s = time.Now().UnixNano()
		}
		gc = client.NewLocalClient(s, game.DefaultWidth, game.DefaultHeight)
	}

	if err := gc.Connect(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
		os.Exit(1)
	}
	snap, err := gc.NextSnapshot()
	if err != nil || snap == nil {
		fmt.Fprintln(os.Stderr, "Erreur: pas de snapshot initial reçu")
		os.Exit(1)
	}

	m := tui.New(gc, snap, *connect != "")
	if err := m.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
		os.Exit(1)
	}
}
