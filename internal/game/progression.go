package game

// XPNeededForLevel renvoie l'XP cumulée nécessaire pour atteindre le niveau
// donné. Paliers : 20, 50, 100, 170, 260... (somme de 20*n).
func XPNeededForLevel(level int) int {
	total := 0
	for n := 1; n < level; n++ {
		total += 20 * n
	}
	return total
}

// EffectiveAttack renvoie l'attaque du joueur, arme équipée incluse.
func (p *Player) EffectiveAttack() int { return p.Attack + p.WeaponBonus }

// EffectiveDefense renvoie la défense du joueur, armure équipée incluse.
func (p *Player) EffectiveDefense() int { return p.Defense + p.ArmorBonus }

// GainXP ajoute de l'XP au joueur et applique les montées de niveau :
// +8 PV max, +8 PV (soin partiel), +1 attaque, +1 défense, +4 mana max.
func (p *Player) GainXP(s *State, amount int) {
	p.XP += amount
	for p.Level < 20 && p.XP >= XPNeededForLevel(p.Level+1) {
		p.Level++
		p.MaxHP += 8
		p.HP += 8
		p.Attack++
		p.Defense++
		p.MaxMana += 4
		p.Mana += 4
		s.Logf("✨ Niveau %d ! PV max %d, ATK %d, DEF %d.", p.Level, p.MaxHP, p.EffectiveAttack(), p.EffectiveDefense())
	}
}
