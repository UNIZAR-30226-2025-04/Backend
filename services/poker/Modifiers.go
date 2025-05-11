package poker

import (
	"fmt"
	"log" // DELETE
	"math/rand"
)

type Modifier struct {
	Value    int `json:"value"`
	LeftUses int `json:"expires_at"`
}

type Modifiers struct {
	Modificadores []Modifier
}

type ReceivedModifier struct {
	Modifier Modifier `json:"modifier"`
	Sender   string   `json:"sender"`
}

type ReceivedModifiers struct {
	Received []ReceivedModifier `json:"modifiers"`
}

type ModifierFunc func(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int)

var modifierTable = map[int]ModifierFunc{
	1: Damn,
	2: PabloHoney,
	3: RAM,
	4: Weezer,
	5: Blonde,
	6: AbbeyRoad,
	7: RockTransgresivo,
	8: DiamondEyes,
	9: TheMoneyStore,
}

var ModifierWeights = []struct {
	ID     int
	Weight int
}{
	{1, 30}, // Most common: 30% chance
	{2, 27}, // Common: 27% chance
	{3, 23}, // Uncommon: 23% chance
	{4, 20}, // Rare: 20% chance
}

// Divide starting chips and mult by 2. 1 round duration
func Damn(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	fichas = fichas / 2
	mult = mult / 2
	return fichas, mult, gold, leftUses
}

// Eern 1 dollar for each card played. 1 round duration
func PabloHoney(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	log.Println("[PABLO-HONEY] gold before:", gold, "cards:", len(hand.Cards))
	gold += len(hand.Cards)
	log.Println("[PABLO-HONEY] gold after:", gold)
	return fichas, mult, gold, leftUses
}

// multiply the chips by random number between 1 and 3
func RAM(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	return fichas*rand.Intn(3) + 1, mult, gold, leftUses
}

// Bans up to 4 players to play four of a kind for 1 round
func Weezer(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	_, _, mano, _ := BestHand(hand)
	if mano == 6 {
		mult = 0
		fichas = 0
	}
	return fichas, mult, gold, leftUses
}

// Bans up to 2 players from playing straight for 1 round
func Blonde(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	_, _, mano, _ := BestHand(hand)
	if mano == 9 {
		mult = 0
		fichas = 0
	}
	return fichas, mult, gold, leftUses
}

// Every King or Queen played scores negatie points. Choose 4 players for 1 round
func AbbeyRoad(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	if leftUses > 0 {
		for _, card := range hand.Cards {
			rank := grade(card)
			if rank == 12 || rank == 13 {
				mult -= 14
			}
		}
	}
	return fichas, mult, gold, leftUses
}

// Aces and K's score double
func RockTransgresivo(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	if leftUses > 0 {
		for _, card := range hand.Cards {
			rank := grade(card)
			if rank == 13 || rank == 14 {
				mult *= 2
			}
		}
	}
	return fichas, mult, gold, leftUses
}

// Applicable to up to 3 players. Substracts from their mult the money the have
func DiamondEyes(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	return fichas, mult - gold, gold, leftUses
}

// Each black card played (spades and clubs) grants 1 dollar, +10 chips, +2 mult. 1 round duration
func TheMoneyStore(hand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	for _, card := range hand.Cards {
		if card.Suit == "s" || card.Suit == "c" {
			gold++
			fichas += 10
			mult += 2
		}
	}
	return fichas, mult, gold, leftUses
}

func Apply(modifier Modifier, hand Hand, fichas int, mult int, gold int) (int, int, int, int) {
	if modifierFunc, exists := modifierTable[modifier.Value]; exists {
		return modifierFunc(hand, modifier.LeftUses, fichas, mult, gold)
	}
	fmt.Printf("Warning: Unknown joker ID — what is %d?\n", modifier.Value)
	return fichas, mult, gold, modifier.LeftUses
}

// Modifiers at each play
func ApplyModifiers(hand Hand, ms *Modifiers, initialFichas int, initialMult int, currentGold int) (int, int, int) {
	currentFichas, currentMult, currentGold := initialFichas, initialMult, currentGold
	finalFichas := initialFichas
	finalMult := initialMult
	finalGold := currentGold
	log.Println("[APPLY-MODIFIERS] gold:", finalGold)

	for _, modifierID := range ms.Modificadores {
		if modifierID.Value == 0 {
			continue
		}
		if modifierID.Value == 2 || modifierID.Value == 4 || modifierID.Value == 3 || modifierID.Value == 5 || modifierID.Value == 6 ||
			modifierID.Value == 7 || modifierID.Value == 8 || modifierID.Value == 9 {
			currentFichas, currentMult, currentGold, modifierID.LeftUses = Apply(modifierID, hand, currentFichas, currentMult, currentGold)
		}
		log.Println("[APPLY-MODIFIERS] Gold obtained from:", modifierID.Value, ":", currentGold)
		finalFichas += currentFichas
		finalMult += currentMult
		finalGold += currentGold
		log.Println("[APPLY-MODIFIERS] Gold after:", modifierID.Value, ":", currentGold)
	}

	return finalFichas, finalMult, finalGold
}

// Modifiers at the begining of the round
func ApplyRoundModifiers(ms *Modifiers, currentGold int) int {
	finalGold := currentGold
	for _, modifierID := range ms.Modificadores {
		if modifierID.Value == 0 {
			continue
		}
		if modifierID.Value == 1 {
			emptyHand := Hand{}
			_, _, currentGold, modifierID.LeftUses = Apply(modifierID, emptyHand, 0, 0, currentGold)
		}
		finalGold += currentGold

	}

	return finalGold
}
