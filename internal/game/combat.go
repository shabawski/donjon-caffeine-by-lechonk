package game

// Attack résout une attaque au corps à corps.
// Base = max(1, attaque - défense), variance de ±20%, coup critique (x2) avec
// 10% de chance. Les dégâts sont appliqués sur le défenseur.
// Retourne les dégâts infligés, si c'était un critique, et si la cible meurt.
func Attack(rng *RNG, attacker, defender *Entity) (dmg int, crit bool, killed bool) {
	base := attacker.Attack - defender.Defense
	if base < 1 {
		base = 1
	}
	variance := 1 + (rng.Float()*0.4 - 0.2)
	dmg = int(float64(base) * variance)
	if dmg < 1 {
		dmg = 1
	}
	crit = rng.Chance(0.10)
	if crit {
		dmg *= 2
	}
	defender.HP -= dmg
	if defender.HP < 0 {
		defender.HP = 0
	}
	return dmg, crit, defender.IsDead()
}
