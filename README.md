# ☕ Le Donjon Caféiné

Un **roguelike dungeon-crawler en terminal**, écrit en Go pur (BubbleTea + Lipgloss).

> Le Seigneur des Tasses, boss du 10e étage, a volé **tout le café** de l'équipe.
> Descends les 10 étages du donjon, explore, combats des rats géants, des gobelins
> et des squelettes, améliore ton héros… et affronte le boss final pour récupérer
> le café. ☕

## Modes de jeu

Deux modes, choisis au menu (solo) ou par le serveur (multi) :

- **COOP (donjon)** — le mode classique : 10 étages, escalier bidirectionnel,
  le boss du dernier étage a volé le café. **Victoire = tuer le boss.**
- **MMORPG (monde ouvert)** — un **overworld** (plaine ouverte, sans escaliers)
  parsemé de **bâtiments ⌂** (portails). Le bâtiment « Donjon » mène à une
  **instance de donjon COOP** (10 étages + boss), créée à la demande et
  réutilisée. Architecture extensible : biomes, PNJ, quêtes, autres portails.

En multijoueur, le mode est choisi au lancement du serveur (`--mode mmorpg`).

## Lancer le jeu

```sh
go run .                 # partie solo (menu : COOP, MMORPG, multi, scores)
go run . --seed 42       # partie solo reproductible (même graine = même donjon)
go run . --connect host:port   # multijoueur : connexion à un serveur distant (TCP)
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

### Serveur dédié (TCP) — multijoueur

```sh
go run ./cmd/coffee-dungeon-server                  # écoute sur :27015 (mode COOP)
go run ./cmd/coffee-dungeon-server --addr :8080     # port personnalisé
go run ./cmd/coffee-dungeon-server --seed 42        # partie reproductible
go run ./cmd/coffee-dungeon-server --mode mmorpg    # monde ouvert (overworld + donjons)
```

Puis connecte-toi depuis un autre terminal / machine (menu → **Multijoueur (TCP)**,
ou en CLI) :

```sh
go run . --connect <host>:27015
```

**Multijoueur** : plusieurs joueurs partagent le même donjon. Chaque joueur a
son **propre champ de vision** (FOV) — le serveur envoie à chaque client un
snapshot personnalisé. Les monstres gèrent leur **focus** côté serveur : ils
poursuivent le joueur le plus proche détecté (portée + ligne de vue).
Les autres joueurs apparaissent sur la carte avec le glyphe `¶`.

> Pour le déploiement sur un VPS : compile le serveur (`go build -o coffee-dungeon-server ./cmd/coffee-dungeon-server`),
> lance-le sur le VPS, ouvre le port (27015) dans le firewall, et connecte les
> clients avec `--connect <ip-vps>:27015` ou via le menu « Multijoueur (TCP) ».

## Commandes

| Touche | Action |
|---|---|
| `ZQSD` / `HJKL` / flèches | se déplacer (attaquer un monstre adjacent en te déplaçant dessus) |
| `.` (point) | attendre un tour |
| `f` | lancer une boule de feu (8 mana, dégâts de zone autour de la cible) |
| `g` ou `>` | descendre l'escalier (se tenir sur la case `>`) |
| `G` ou `<` | remonter l'escalier (se tenir sur la case `<`) |
| `e` | entrer dans un bâtiment `⌂` adjacent (MMORPG, monde ouvert) |
| `x` | sortir du donjon vers le monde ouvert (MMORPG) |
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

Options : `--max-turns 30000` (limite de tours, défaut 30000), `--mode mmorpg`
(démo MMORPG : le bot entre dans le premier donjon de l'overworld puis joue),
`--help` (usage).

## Tests

```sh
go test ./...
```

- `internal/game` : tests unitaires (RNG déterministe, génération procédurale et
  connexité, FOV/fog of war, combat, IA, progression, transitions victoire/défaite,
  dimensions d'étage `NewStateSized`/`ClampWorldSize`, registre ECS/SOA `Entities`,
  équilibrage : templates monstres, rollXP borné, Ascend, étages stables RNG dérivé,
  **architecture maps** : `MapRegistry`/`Root`, overworld MMORPG, entrée/sortie
  des bâtiments (`EnterBuilding`/`ExitBuilding`, instance réutilisée), fix bug
  maps (map parente non affectée), focus IA restreint à la map courante,
  victoire MMORPG de bout en bout)
  + tests d'intégration (parties complètes en bot sur 20 graines COOP + 5 graines
  MMORPG, victoire et défaite de bout en bout).
- `internal/score` : persistance (ajout, tri, top 10, fichier absent/corrompu).
- `internal/tui` : smoke tests (rendu, navigation clavier, sauvegarde du score)
  + tests du viewport plein écran (caméra centrée sur le joueur, resize,
  hauteur de vue = hauteur du terminal).

## Structure du projet

```
main.go               point d'entrée client (CLI : --seed, --demo, --mode, --connect, --max-turns)
cmd/coffee-dungeon-server/   serveur de jeu dédié (TCP, déploiement VPS, --mode coop|mmorpg)
internal/game/        moteur de jeu pur — AUCUNE dépendance TUI, testé unitairement
  rng.go              RNG déterministe (seedable)
  world.go            génération procédurale d'étage (salles + couloirs + escaliers >/<)
  map.go              Map (instance d'étage) + MapRegistry (registre de maps, illimité)
  overworld.go        génération de l'overworld MMORPG (plaine + bâtiments ⌂ portails)
  building.go         entrée/sortie des bâtiments (instances de donjon à la demande)
  fov.go              champ de vision (rayons de Bresenham) + fog of war
  ecs.go              registre ECS/SOA des monstres (tableaux parallèles par composant)
  entity.go           joueur, monstres (7 types), boss (stats, glyphes, templates)
  combat.go           résolution d'attaque (défense, variance, critiques)
  ai.go               IA ennemie + pathfinding BFS + détection par ligne de vue (par map)
  items.go            objets (or, potions, armes, armures) + ItemValue (prix marchand)
  merchant.go         PNJ marchand (spawn étages pairs, boutique, achat/vente)
  progression.go      XP (paliers 30*n, variance ±20%), niveaux, attaque/défense effectives
  state.go            état de partie, tours, sorts, victoire/défaite, score, journal, Ascend()
  auto.go             bot AutoPlay/AutoPlayMMORPG (mode --demo et tests d'intégration)
  snapshot.go         Snapshot sérialisable (JSON) — état visible envoyé aux clients
internal/net/         protocole client/serveur (enveloppes JSON : Hello/Welcome/Input/Snapshot/End)
internal/client/      GameClient (interface) + LocalClient (solo, channels) + TCPClient (multi, TCP)
internal/server/      serveur de jeu TCP (détient le game.State, applique les inputs, publie les snapshots)
internal/tui/         interface terminal (BubbleTea + Lipgloss), plein écran — rend depuis un Snapshot
  model.go update.go view.go   modèle MVC BubbleTea + viewport (caméra centrée)
  keys.go             ZQSD/HJKL/flèches → déplacements
  styles.go           styles Lipgloss (variables package-level)
  menu.go             menu principal + tableau des scores
  overlays.go         aide (?) et inventaire (i) avec liste d'objets
  shop.go             écran marchand (achat/vente, onglets)
  gameover.go         écrans victoire / game over
internal/score/       persistance des high scores (JSON, top 10, écriture atomique)
```

### Architecture client/serveur

Le TUI ne touche **plus** au moteur de jeu directement. Il communique avec un
serveur via l'interface `client.GameClient` :

- **Solo** (`LocalClient`) : le serveur tourne dans une goroutine du même
  process et communique par **go channels**. Aucune différence visible pour
  le joueur.
- **Multijoueur** (`TCPClient`) : connexion **TCP** à `cmd/coffee-dungeon-server`.
  Le client envoie des inputs (JSON), le serveur applique sur son `game.State`
  (autorité) et renvoie des **snapshots**.

Le serveur détient l'état autoritaire (`game.State`). Le client reçoit des
`game.Snapshot` (capture sérialisable JSON de l'état visible : carte, joueur,
monstres, FOV, journal) et rend depuis ce snapshot. Le protocole est du JSON
délimités par des sauts de ligne (`internal/net/proto.go`).

En **multijoueur**, le serveur gère plusieurs joueurs (`players map[int]*game.Player`)
avec un **FOV par joueur** : chaque client reçoit un snapshot personnalisé
(position + champ de vision). Les monstres gèrent leur **focus** côté serveur
(`focusPlayer` dans `internal/game/ai.go`) : ils ciblent le joueur le plus
proche détecté **sur leur propre map** (portée + ligne de vue).

### Architecture du moteur : maps (v3)

Un **étage = une `Map`** (instance complète : monde, monstres, items, marchand,
FOV par joueur). Plusieurs maps coexistent dans le serveur
(`State.Maps *MapRegistry`, nombre illimité, créées à la demande) :

- Chaque joueur a un **`MapID`** : il ne voit et n'agit que dans sa map.
- **Descendre/monter** change de `MapID` : la map parente persiste dans le
  registre, les autres maps (et les joueurs dessus) ne sont **pas affectés**
  (fix du bug maps 17/10 17:07). Remonter retourne sur la **même instance**
  (mêmes tuiles, pas de régénération).
- **MMORPG** : la map racine est l'**overworld** (`MapKindOverworld`, plaine
  sans escaliers) parsemée de **bâtiments ⌂** (`Building`). Un bâtiment
  « Donjon » mène à une **instance de donjon** (`MapKindBuilding`), créée à
  la demande et **réutilisée** (une seule par bâtiment, liée via
  `Building.InstanceMapID`). Touches : `e` pour entrer, `x` pour sortir.
- Les monstres ne voient/agissent que sur leur map : le **focus IA** est
  restreint aux joueurs présents sur la même map.
- Les étages sont générés avec un **RNG dérivé** (`seed*1000 + mapID`) :
  déterministes et stables entre les visites.

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
