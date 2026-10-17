# AGENTS.md — Donjon Caféiné

Roguelike TUI en Go. Contexte, procédures et frontières à respecter à chaque tâche.

## Procédures (ordre de validation)

```sh
gofmt -l .        # doit être vide
go build ./...    # doit passer
go vet ./...      # doit passer
go test ./...     # doit passer (unitaires + intégration + smoke TUI)
go run . --demo --seed 42   # partie headless de bout en bout, sans panique
```

- Après tout ajout d'import : `go mod tidy`.
- Jamais de `log.Fatal` hors de `cmd/` ; pas de `panic` dans le moteur.

## Frontières (strictes)

- **`internal/game/`** : logique de jeu **pure**. Interdit d'importer
  `bubbletea`, `lipgloss` ou tout paquet TUI. Tout comportement y est testable
  unitairement. Le moteur est **déterministe** : même seed = même partie.
- **`internal/tui/`** : interface BubbleTea uniquement. `Update()` est
  déterministe et sans effet de bord (pas d'I/O, pas de sleep) ; `View()` ne
  mutue jamais l'état. Les styles Lipgloss sont des **variables package-level**
  (`styles.go`), jamais recréés dans `View()`.
- **`internal/score/`** : persistance JSON, top 10, écriture atomique
  (temp + rename) dans `os.UserConfigDir()/cafedungeon/scores.json`.

## Conventions

- Un fichier par responsabilité ; un composant TUI par fichier.
- Touches : ZQSD (AZERTY) + HJKL + flèches. **Sur AZERTY `q` = gauche** ;
  quitter = `Q` (maj) ou `esc`. MMORPG : `e` = entrer dans un bâtiment ⌂,
  `x` = sortir du donjon vers le monde ouvert.
- Journal du jeu : `State.Logf`, plafonné à `MaxLog` (200) entrées.
- Un test doit accompagner tout nouveau comportement du moteur (génération, FOV,
  combat, IA, progression, transitions de phase, maps/bâtiments).

## Modes de jeu (v3)

- **COOP** (défaut) : donjon 10 étages, victoire = tuer le boss.
- **MMORPG** : overworld (map racine, sans escaliers) + bâtiments ⌂ portails
  vers des instances de donjon créées à la demande (`Building.InstanceMapID`).
  Le mode se choisit au menu (solo) ou via `--mode` (serveur TCP / `--demo`).

## Architecture maps (v3)

- Un **étage = une `Map`** (monde + entités + items + marchand + FOV par
  joueur) ; plusieurs maps coexistent dans `State.Maps *MapRegistry` (illimité).
- Chaque joueur a un **`MapID`** ; descendre/monter change de `MapID`, la map
  parente persiste et les autres maps/joueurs ne sont **pas affectés**.
- Remonter retourne sur la **même instance** (pas de régénération) :
  `enterMap` utilise `cur.ParentMapID` en remontée (`NoParentMapID = -1`).
- Le **focus IA** est restreint à la map courante (`playersOnCurMap`).

## Livraison

Critère de finition d'une tâche sur ce dépôt : `go build ./...` OK,
`go vet ./...` OK, `gofmt -l .` vide, `go test ./...` OK, démo headless
(`go run . --demo --seed 42`) terminée sans panique, README.md à jour.

## Rappels de conception

- Le boss du dernier étage (10) a volé le café : victoire = le tuer.
- Dernier étage : pas d'escalier (`World.HasStairs == false`), le boss y campe.
- Le bot `AutoPlay`/`AutoPlayMMORPG` + `AutoAction` sert au mode `--demo`
  (COOP et MMORPG) ET aux tests d'intégration : ne pas le réserver au CLI.
