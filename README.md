# ☕ Le Donjon Caféiné

Un **roguelike dungeon-crawler en terminal**, écrit en Go pur (BubbleTea + Lipgloss).

> Le Seigneur des Tasses, boss du 10e étage, a volé **tout le café** de l'équipe.
> Descends les 10 étages du donjon, explore, combats des rats géants, des gobelins
> et des squelettes, améliore ton héros… et affronte le boss final pour récupérer
> le café. ☕

## Lancer le jeu

```sh
go run .                 # partie jouable (interface terminal plein écran)
go run . --seed 42       # partie reproductible (même graine = même donjon)
```

L'interface occupe **tout le terminal** (plein écran alternatif) et s'adapte
automatiquement quand tu redimensionnes la fenêtre : la carte est rendue via
un viewport centré sur le joueur, le panneau HUD à droite ajuste son journal
à la hauteur disponible.

Ou compiler un binaire :

```sh
go build -o donjon-cafeine .
./donjon-cafeine
```

## Commandes

| Touche | Action |
|---|---|
| `ZQSD` / `HJKL` / flèches | se déplacer (attaquer un monstre adjacent en te déplaçant dessus) |
| `.` (point) | attendre un tour |
| `f` | lancer une boule de feu (8 mana, dégâts de zone autour de la cible) |
| `g` ou `>` | descendre l'escalier (se tenir sur la case `>`) |
| `G` ou `<` | remonter l'escalier (se tenir sur la case `<`) |
| `i` | inventaire (équiper `e`, utiliser `u`, jeter `d`) |
| `m` | marchand (se tenir près du PNJ `M`) |
| `?` | aide intégrée |
| `Q` (majuscule) ou `esc` | retour au menu (abandon de la partie) |
| `r` | rejouer (sur l'écran de fin) |
| `q` | quitter (depuis le menu) |

> Sur clavier AZERTY, `q` sert au déplacement (gauche) : quitter se fait avec `Q` ou `esc`.

## Règles du jeu

- **10 étages** générés procéduralement (salles reliées par des couloirs), du plus
  facile au plus profond. **Escalier bidirectionnel** : `>` pour descendre, `<` pour
  remonter (les étages sont déterministes : remonter puis redescendre régénère la
  même carte).
- **Fog of war** : tu ne vois que les cases dans ta portée de vue (8 cases) ;
  le reste de ce que tu as exploré reste visible en atténué, le reste est noir.
- **Combat au tour par tour** : dégâts = `max(1, attaque − défense)`, variance de
  ±20 %, **coup critique** (×2) avec 10 % de chance.
- **Monstres** (7 types répartis par profondeur) : rat géant (fuit quand il est
  blessé), **slime** (fuit tôt), gobelin, **chauve-souris** (détection longue),
  squelette, **golem de pierre** (très résistant, ne fuit jamais) — et le
  **Seigneur des Tasses** au 10e étage (120 PV, 11 ATK, 5 DEF, rage sous 50 % PV),
  qui te flaire dans un rayon de 12 cases même sans te voir.
- **Détection des monstres** : un monstre te détecte s'il est à portée **et** a une
  ligne de vue dégagée sur toi (les murs bloquent la détection). Le boss fait
  exception : il te flaire à travers les murs.
- **IA ennemie** : patrouille quand le joueur n'est pas détecté, poursuite par
  plus court chemin (BFS) quand il l'est, fuite pour les lâches.
- **Objets** : au sol, ramassés en marchant dessus. L'or est crédité directement ;
  les **armes, armures et potions vont dans l'inventaire** (pas d'auto-équip).
  Depuis l'inventaire (`i`) : `e` équiper, `u` utiliser une potion, `d` jeter.
- **Marchand** : un PNJ `M` campe sur les **étages pairs** (hors dernier étage).
  Touche `m` à côté de lui pour ouvrir sa boutique : onglet **Achat** (armes,
  armures, potions adaptées à la profondeur) et onglet **Vente** (revente à 50%
  de la valeur).
- **Progression** : XP à chaque kill avec **variance ±20 %** (min-max déterministe),
  paliers de niveau à `30×n` XP cumulée, montée de niveau (max 20) : +10 PV max,
  +1 ATK, +1 DEF, +4 mana max. PV de départ : 40.
- **Mana** : la boule de feu coûte 8 mana ; +1 mana tous les 3 tours.
- **Victoire** : tuer le Seigneur des Tasses au 10e étage.
- **Défaite** : PV à 0.

## Score

```
score = or×2 + kills×10 + niveau×25 + étage×50 + 1000 (si victoire)
```

Le **top 10** est persisté entre les lancements dans
`~/.config/cafedungeon/scores.json` (visible depuis le menu : « Meilleurs scores »).

## Vérification headless (CI / démo)

Une partie complète est jouée automatiquement par un bot, sans terminal :

```sh
go run . --demo --seed 42
# [VICTOIRE] seed=42 étage=10 niveau=7 PV=62/78 or=318 kills=32 tours=13457 score=2631
```

Options : `--max-turns 30000` (limite de tours, défaut 30000), `--help` (usage).

## Tests

```sh
go test ./...
```

- `internal/game` : tests unitaires (RNG déterministe, génération procédurale et
  connexité, FOV/fog of war, combat, IA, progression, transitions victoire/défaite,
  dimensions d'étage `NewStateSized`/`ClampWorldSize`, registre ECS/SOA `Entities`,
  équilibrage : templates monstres, rollXP borné, Ascend, étages stables RNG dérivé)
  + tests d'intégration (parties complètes en bot sur 20 graines, victoire et
  défaite de bout en bout).
- `internal/score` : persistance (ajout, tri, top 10, fichier absent/corrompu).
- `internal/tui` : smoke tests (rendu, navigation clavier, sauvegarde du score)
  + tests du viewport plein écran (caméra centrée sur le joueur, resize,
  hauteur de vue = hauteur du terminal).

## Structure du projet

```
main.go               point d'entrée (CLI : --seed, --demo, --max-turns)
internal/game/        moteur de jeu pur — AUCUNE dépendance TUI, testé unitairement
  rng.go              RNG déterministe (seedable)
  world.go            génération procédurale d'étage (salles + couloirs + escaliers >/<)
  fov.go              champ de vision (rayons de Bresenham) + fog of war
  ecs.go              registre ECS/SOA des monstres (tableaux parallèles par composant)
  entity.go           joueur, monstres (7 types), boss (stats, glyphes, templates)
  combat.go           résolution d'attaque (défense, variance, critiques)
  ai.go               IA ennemie + pathfinding BFS + détection par ligne de vue
  items.go            objets (or, potions, armes, armures) + ItemValue (prix marchand)
  merchant.go         PNJ marchand (spawn étages pairs, boutique, achat/vente)
  progression.go      XP (paliers 30*n, variance ±20%), niveaux, attaque/défense effectives
  state.go            état de partie, tours, sorts, victoire/défaite, score, journal, Ascend()
  auto.go             bot AutoPlay (mode --demo et tests d'intégration, farm par étage)
internal/tui/         interface terminal (BubbleTea + Lipgloss), plein écran
  model.go update.go view.go   modèle MVC BubbleTea + viewport (caméra centrée)
  keys.go             ZQSD/HJKL/flèches → déplacements
  styles.go           styles Lipgloss (variables package-level)
  menu.go             menu principal + tableau des scores
  overlays.go         aide (?) et inventaire (i) avec liste d'objets
  shop.go             écran marchand (achat/vente, onglets)
  gameover.go         écrans victoire / game over
internal/score/       persistance des high scores (JSON, top 10, écriture atomique)
```

### Architecture du moteur : ECS + SOA

Les monstres sont stockés dans un **registre ECS** (`internal/game/ecs.go`) avec
une organisation **SOA** (Structure of Arrays) : chaque composant (position,
PV, attaque, etc.) est un tableau parallèle indexé par `EntityID`, plutôt qu'un
tableau de structs. Avantages :

- **Localité cache** : les itérations de l'IA parcourent des tranches contiguës
  (`xs`, `ys`, `hps`…), sans indirection pointeur.
- **Pas d'allocation par entité** : un spawn est un `append` sur chaque tableau.
- **Compaction** : les monstres morts sont supprimés physiquement en fin de tour
  (`Compact`), libérant la RAM et gardant les tableaux denses.
- **Sérialisation naturelle** : les tableaux parallèles se prêtent au protocole
  réseau (client/serveur) prévu en v2.

Le joueur reste une struct à part entière (`Player` embarque un `Entity`) ;
les objets au sol restent un slice (peu nombreux, pas de hot path).

## Dépendances

- [BubbleTea](https://github.com/charmbracelet/bubbletea) — boucle TUI
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — styles terminal

Aucune dépendance pour le moteur de jeu (`internal/game`) : il est pur Go stdlib.
