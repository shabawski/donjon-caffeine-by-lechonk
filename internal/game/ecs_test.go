package game

import "testing"

func TestEntitiesSpawnAndLen(t *testing.T) {
	e := NewEntities()
	if e.Len() != 0 {
		t.Fatalf("registre vide attendu, got %d", e.Len())
	}
	id1 := e.Spawn(enemyTemplate(KindRat))
	id2 := e.Spawn(enemyTemplate(KindGoblin))
	if e.Len() != 2 {
		t.Fatalf("2 entités attendues, got %d", e.Len())
	}
	if id1 == id2 {
		t.Fatal("les IDs doivent être distincts")
	}
	if !e.IsAlive(id1) || !e.IsAlive(id2) {
		t.Fatal("les entités spawnées doivent être vivantes")
	}
}

func TestEntitiesDespawnAndCompact(t *testing.T) {
	e := NewEntities()
	id1 := e.Spawn(enemyTemplate(KindRat))
	id2 := e.Spawn(enemyTemplate(KindGoblin))
	e.Spawn(enemyTemplate(KindSkeleton)) // index 2, sera compacté vers l'index 1
	e.Despawn(id2)
	if e.Len() != 2 {
		t.Fatalf("après Despawn: 2 vivantes attendues, got %d", e.Len())
	}
	if e.IsAlive(id2) {
		t.Fatal("l'entité despawnée ne doit plus être vivante")
	}
	e.Compact()
	if e.Len() != 2 {
		t.Fatalf("après Compact: 2 entités, got %d", e.Len())
	}
	// Les entités vivantes conservent leurs données après compaction.
	if e.Kind[int(id1)] != KindRat {
		t.Fatalf("id1 perd son Kind après Compact: %v", e.Kind[int(id1)])
	}
	// id3 a été décalé vers l'index 1 par la compaction.
	if e.Kind[1] != KindSkeleton {
		t.Fatalf("entité survivante mal compactée: index 1 = %v", e.Kind[1])
	}
	// L'ancien index 2 est hors limites.
	if len(e.Alive) != 2 {
		t.Fatalf("tableaux non tronqués après Compact: %d", len(e.Alive))
	}
}

func TestEntitiesEnemyAt(t *testing.T) {
	e := NewEntities()
	id := e.Spawn(enemyTemplate(KindRat))
	e.X[int(id)], e.Y[int(id)] = 5, 7
	if got := e.EnemyAt(5, 7); got != id {
		t.Fatalf("EnemyAt(5,7) = %v, attendu %v", got, id)
	}
	if got := e.EnemyAt(1, 1); got != InvalidEntity {
		t.Fatalf("EnemyAt(1,1) = %v, attendu InvalidEntity", got)
	}
	// Un monstre mort n'est plus trouvé.
	e.Despawn(id)
	if got := e.EnemyAt(5, 7); got != InvalidEntity {
		t.Fatalf("EnemyAt après Despawn = %v, attendu InvalidEntity", got)
	}
}

func TestEntitiesBossID(t *testing.T) {
	e := NewEntities()
	if got := e.BossID(); got != InvalidEntity {
		t.Fatalf("pas de boss attendu, got %v", got)
	}
	e.Spawn(enemyTemplate(KindRat))
	boss := e.Spawn(enemyTemplate(KindBoss))
	if got := e.BossID(); got != boss {
		t.Fatalf("BossID = %v, attendu %v", got, boss)
	}
	e.Despawn(boss)
	if got := e.BossID(); got != InvalidEntity {
		t.Fatalf("boss mort: BossID = %v, attendu InvalidEntity", got)
	}
}

func TestEntitiesSnapshot(t *testing.T) {
	e := NewEntities()
	tmpl := enemyTemplate(KindGoblin)
	tmpl.X, tmpl.Y = 3, 4
	id := e.Spawn(tmpl)
	snap := e.Snapshot(id)
	if snap.Kind != KindGoblin || snap.X != 3 || snap.Y != 4 || snap.HP != 14 {
		t.Fatalf("snapshot incorrect: %+v", snap)
	}
	// Modifier le snapshot ne modifie pas le registre.
	snap.HP = 1
	if e.HP[int(id)] != 14 {
		t.Fatal("le snapshot doit être une copie, pas une vue")
	}
}

func TestEntitiesResetKeepsCapacity(t *testing.T) {
	e := NewEntities()
	for i := 0; i < 5; i++ {
		e.Spawn(enemyTemplate(KindRat))
	}
	capBefore := cap(e.Alive)
	e.Reset()
	if e.Len() != 0 {
		t.Fatalf("registre non vide après Reset: %d", e.Len())
	}
	if cap(e.Alive) != capBefore {
		t.Fatal("Reset doit conserver la capacité des tableaux (pas de réallocation)")
	}
}

func TestEntitiesEachAliveOrder(t *testing.T) {
	e := NewEntities()
	id1 := e.Spawn(enemyTemplate(KindRat))
	id2 := e.Spawn(enemyTemplate(KindGoblin))
	e.Spawn(enemyTemplate(KindSkeleton))
	e.Despawn(id2)
	var visited []EntityID
	e.EachAlive(func(id EntityID) { visited = append(visited, id) })
	if len(visited) != 2 {
		t.Fatalf("2 entités vivantes attendues, got %d", len(visited))
	}
	if visited[0] != id1 {
		t.Fatalf("ordre de parcours incorrect: premier = %v, attendu %v", visited[0], id1)
	}
}

func TestEntitiesAllSnapshots(t *testing.T) {
	e := NewEntities()
	e.Spawn(enemyTemplate(KindRat))
	e.Spawn(enemyTemplate(KindGoblin))
	e.Spawn(enemyTemplate(KindSkeleton))
	e.Despawn(EntityID(1))
	snaps := e.AllSnapshots()
	if len(snaps) != 2 {
		t.Fatalf("2 snapshots attendus, got %d", len(snaps))
	}
	if snaps[0].Kind != KindRat || snaps[1].Kind != KindSkeleton {
		t.Fatalf("snapshots incorrects: %v, %v", snaps[0].Kind, snaps[1].Kind)
	}
}
