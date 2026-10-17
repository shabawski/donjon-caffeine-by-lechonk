// Command coffee-dungeon-server est le serveur de jeu TCP du Donjon Caféiné.
// Il détient l'état autoritaire de la partie, reçoit les actions des clients
// connectés en TCP, les applique, et publie des snapshots (JSON newline-delimited).
//
// Le client se connecte avec : donjon-cafeine --connect host:port
//
// En solo, pas besoin de ce serveur : le client embarque le jeu via channels
// (client.LocalClient). Ce serveur sert pour le multijoueur (déploiement VPS).
//
// Usage :
//
//	coffee-dungeon-server                  écoute sur :27015
//	coffee-dungeon-server --addr :8080     écoute sur le port 8080
//	coffee-dungeon-server --seed 42        partie reproductible
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"game-test/internal/game"
	"game-test/internal/server"
)

func main() {
	addr := flag.String("addr", ":27015", "adresse d'écoute TCP (host:port)")
	seed := flag.Int64("seed", 0, "graine de la partie (0 = aléatoire)")
	flag.Parse()

	s := *seed
	if s == 0 {
		s = time.Now().UnixNano()
	}

	srv := server.NewServer(s, game.DefaultWidth, game.DefaultHeight)
	if err := srv.Listen(*addr); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur listen:", err)
		os.Exit(1)
	}
	fmt.Printf("☕ Serveur Donjon Caféiné — écoute sur %s (seed=%d)\n", srv.Addr(), s)
	fmt.Println("   Connecte-toi avec : donjon-cafeine --connect", srv.Addr())
	if err := srv.Serve(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur serve:", err)
		os.Exit(1)
	}
}
