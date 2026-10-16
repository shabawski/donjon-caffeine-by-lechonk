package tui

// dirFromKey traduit une touche en déplacement (dx, dy).
// Supporte les flèches, HJKL (vim) et ZQSD (clavier AZERTY).
func dirFromKey(k string) (dx, dy int, ok bool) {
	switch k {
	case "up", "k", "z", "8":
		return 0, -1, true
	case "down", "j", "s", "2":
		return 0, 1, true
	case "left", "h", "q", "4":
		return -1, 0, true
	case "right", "l", "d", "6":
		return 1, 0, true
	}
	return 0, 0, false
}
