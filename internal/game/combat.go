package game

// Attack résout une attaque au corps à corps entre deux combattants.
// atkAttack et defDefense sont les attaques/défenses effectives (équipement
// inclus côté joueur). Les dégâts sont appliqués sur defenderHP (pointeur
// vers le composant HP du défenseur — registre SOA ou Player.HP).
// Base = max(1, attaque - défense), variance de ±20%, coup critique (x2)
// avec 10% de chance.
// Retourne les dégâts infligés, si c'était un critique, et si la cible meurt.
func Attack(rng *RNG, atkAttack, defDefense int, defenderHP *int) (dmg int, crit bool, killed bool) {
	base := atkAttack - defDefense
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
	*defenderHP -= dmg
	if *defenderHP < 0 {
		*defenderHP = 0
	}
	killed = *defenderHP <= 0
	return dmg, crit, killed
}
