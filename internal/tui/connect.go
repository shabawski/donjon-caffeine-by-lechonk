package tui

import "strings"

// viewConnect rend l'écran de saisie de l'adresse du serveur (host:port)
// pour le mode multijoueur. L'adresse est saisie au clavier, enter connecte.
func (m *Model) viewConnect() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("🌐 Multijoueur — connexion serveur") + "\n\n")
	b.WriteString("   Adresse du serveur (host:port) :\n\n")
	// Champ de saisie avec curseur.
	b.WriteString("   " + styleCursor.Render(m.connectInput+"█") + "\n\n")
	if m.connectErr != "" {
		b.WriteString(styleGameOver.Render("   Erreur : "+m.connectErr) + "\n\n")
	}
	b.WriteString("   Exemple : 127.0.0.1:27015  ou  51.75.123.45:27015\n\n")
	b.WriteString(styleDim.Render("   enter : connecter · esc : retour au menu"))
	return b.String()
}
