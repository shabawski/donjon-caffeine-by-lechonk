package game

// XPNeededForLevel renvoie l'XP cumulée nécessaire pour atteindre le niveau
// donné. Paliers : 30, 90, 180, 300... (somme de 30*n).
// Palier 30 (vs 20 en v1) : progression ralentie pour que le boss final ne
// meure plus en 2 coups après 10 étages de farm.
func XPNeededForLevel(level int) int {
	total := 0
	for n := 1; n < level; n++ {
		total += 30 * n
	}
	return total
}

// EffectiveAttack renvoie l'attaque du joueur, arme équipée incluse.
func (p *Player) EffectiveAttack() int { return p.Attack + p.WeaponBonus }

// EffectiveDefense renvoie la défense du joueur, armure équipée incluse.
func (p *Player) EffectiveDefense() int { return p.Defense + p.ArmorBonus }

// GainXP ajoute de l'XP au joueur et applique les montées de niveau :
// +10 PV max, +10 PV (soin partiel), +1 attaque, +1 défense, +4 mana max.
// (+10 PV vs +8 en v1 : compense le boss renforcé.)
func (p *Player) GainXP(s *State, amount int) {
	p.XP += amount
	for p.Level < 20 && p.XP >= XPNeededForLevel(p.Level+1) {
		p.Level++
		p.MaxHP += 10
		p.HP += 10
		p.Attack++
		p.Defense++
		p.MaxMana += 4
		p.Mana += 4
		s.Logf("✨ Niveau %d ! PV max %d, ATK %d, DEF %d.", p.Level, p.MaxHP, p.EffectiveAttack(), p.EffectiveDefense())
	}
}
