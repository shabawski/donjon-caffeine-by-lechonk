// Package net définit le protocole de communication client/serveur du
// Donjon Caféiné. Les messages sont des enveloppes JSON : un type + un
// payload typé. Le transport est TCP (multi) ou channels (solo, via
// LocalClient qui embarque le même protocole en mémoire).
package net

import (
	"encoding/json"

	"game-test/internal/game"
)

// MsgType identifie le type d'un message échangé.
type MsgType int

const (
	MsgHello    MsgType = iota // client → serveur : connexion (nom du joueur)
	MsgWelcome                 // serveur → client : accueil (id, seed, dimensions)
	MsgInput                   // client → serveur : action du joueur
	MsgSnapshot                // serveur → client : état visible du jeu
	MsgEnd                     // serveur → client : fin de partie
)

// Envelope est l'enveloppe de tout message échangé (JSON).
type Envelope struct {
	Type    MsgType         `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// HelloPayload : connexion d'un client (nom du joueur).
type HelloPayload struct {
	Name string `json:"name"`
}

// WelcomePayload : accueil du serveur (identité du joueur, config de la partie).
type WelcomePayload struct {
	PlayerID int   `json:"playerId"`
	Seed     int64 `json:"seed"`
	Width    int   `json:"width"`
	Height   int   `json:"height"`
}

// InputKind identifie l'action demandée par le client.
type InputKind string

const (
	InputMove     InputKind = "move"     // DX/DY : déplacement ou attaque
	InputWait     InputKind = "wait"     // attendre un tour
	InputFireball InputKind = "fireball" // LastDX/LastDY : direction du sort
	InputDescend  InputKind = "descend"  // descendre l'escalier
	InputAscend   InputKind = "ascend"   // remonter l'escalier
	InputEquip    InputKind = "equip"    // Index : équiper un objet
	InputUse      InputKind = "use"      // Index : utiliser une potion
	InputDrop     InputKind = "drop"     // Index : jeter un objet
	InputBuy      InputKind = "buy"      // Index : acheter au marchand
	InputSell     InputKind = "sell"     // Index : vendre au marchand
)

// InputPayload : action du joueur envoyée au serveur.
type InputPayload struct {
	Kind   InputKind `json:"kind"`
	DX     int       `json:"dx,omitempty"`
	DY     int       `json:"dy,omitempty"`
	Index  int       `json:"index,omitempty"`
	LastDX int       `json:"lastDx,omitempty"`
	LastDY int       `json:"lastDy,omitempty"`
}

// SnapshotPayload : état visible du jeu envoyé au client.
type SnapshotPayload struct {
	Snap *game.Snapshot `json:"snap"`
}

// EndPayload : fin de partie (victoire, défaite, déconnexion).
type EndPayload struct {
	Reason string `json:"reason"`
}

// Encode sérialise un payload dans une enveloppe.
func Encode(t MsgType, payload any) (Envelope, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Type: t, Payload: data}, nil
}

// DecodePayload désérialise le payload d'une enveloppe dans v.
func DecodePayload(env Envelope, v any) error {
	return json.Unmarshal(env.Payload, v)
}
