package poker

import (
	"fmt"
)

func compareHands(hand1, hand2 Hand) bool {
	if len(hand1.Cards) != len(hand2.Cards) {
		return false
	}
	for i := range hand1.Cards {
		if hand1.Cards[i] != hand2.Cards[i] {
			return false
		}
	}
	return true
}

type Modifier struct {
	Value    int `json:"value"`
	LeftUses int `json:"expires_at"` // -1 if no acaba hasta final de partida
}

type Modifiers struct {
	Modificadores []Modifier
}

type ReceivedModifier struct {
	Modifier Modifier `json:"modifier"`
	Sender   string   `json:"sender"`
}

type ModifierFunc func(hand Hand, bestHand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int)

var modifierTable = map[int]ModifierFunc{
	1: EvilEye,
	2: LuckyGlove,
	3: HotStreak,
	4: CoinPurse,
}

var ModifierWeights = []struct {
	ID     int
	Weight int
}{
	{1, 40}, // Most common: 40% chance
	{2, 30}, // Common: 30% chance
	{3, 20}, // Uncommon: 20% chance
	{4, 10}, // Rare: 10% chance
}

// -1 to the multiplier of the target’s most-used hand type.
func EvilEye(hand Hand, bestHand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	if compareHands(hand, bestHand) {
		return fichas, mult - 1, gold, leftUses - 1
	}
	return fichas, mult, gold, leftUses - 1
}

// Gain +2 Gold at the start of each round
func CoinPurse(hand Hand, bestHand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	return fichas, mult, gold + 2, leftUses - 1
}

// +15 Chips to all Flush hands
func LuckyGlove(hand Hand, bestHand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	_, isFlush := Flush(hand)
	if isFlush {
		return fichas + 15, mult, gold, leftUses - 1
	}
	return fichas, mult, gold, leftUses - 1
}

// 2x Multiplier
func HotStreak(hand Hand, bestHand Hand, leftUses int, fichas int, mult int, gold int) (int, int, int, int) {
	return fichas, mult * 2, gold, leftUses - 1
}

func Apply(modifier Modifier, hand Hand, bestHand Hand, fichas int, mult int, gold int) (int, int, int, int) {
	if modifierFunc, exists := modifierTable[modifier.Value]; exists {
		return modifierFunc(hand, bestHand, modifier.LeftUses, fichas, mult, gold)
	}
	fmt.Printf("Warning: Unknown joker ID — what is %d?\n", modifier.Value)
	return fichas, mult, gold, modifier.LeftUses
}

// Modifiers at each play
func ApplyModifiers(hand Hand, bestHand Hand, ms *Modifiers, initialFichas int, initialMult int, currentGold int) (int, int, int) {
	currentFichas, currentMult, currentGold := initialFichas, initialMult, currentGold
	finalFichas := initialFichas
	finalMult := initialMult
	finalGold := currentGold

	for _, modifierID := range ms.Modificadores {
		if modifierID.Value == 0 {
			continue
		}
		if modifierID.Value == 1 || modifierID.Value == 2 || modifierID.Value == 3 {
			currentFichas, currentMult, currentGold, modifierID.LeftUses = Apply(modifierID, hand, bestHand, currentFichas, currentMult, currentGold)
		}
		finalFichas += currentFichas
		finalMult += currentMult
		finalGold += currentGold
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
		if modifierID.Value == 4 {
			emptyHand := Hand{}
			_, _, currentGold, modifierID.LeftUses = Apply(modifierID, emptyHand, emptyHand, 0, 0, currentGold)
		}
		finalGold += currentGold

	}

	return finalGold
}
