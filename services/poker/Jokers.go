package poker

import (
	"fmt"
	"math/rand"
)

type Jokers struct {
	Juglares []int
}

type JokerFunc func(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool)

const MaxJokers = 10

var jokerTable = map[int]JokerFunc{
	1:  SolidSevenJoker,
	2:  PoorJoker,
	3:  BotardoJoker,
	4:  AverageSizeMichel,
	5:  HellCowboy,
	6:  CarbSponge,
	7:  Photograph,
	8:  Petpet,
	9:  EmptyJoker,
	10: TwoFriendsJoker,
	11: LiriliLarila,
	12: BIRDIFICATION,
}

func SolidSevenJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return fichas + 7, mult + 7, gold, used
}

func BotardoJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return 1, 1, 1, used // CHANGE EVENTUALLY
}

func AverageSizeMichel(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	randomNumber := rand.Intn(15) + 1
	if randomNumber == 1 {
		used[index] = true
		return fichas, mult + 15, gold, used
	}
	return fichas, mult, gold, used
}

func PoorJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return gold + 4, fichas, mult, used
}

func CarbSponge(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	_, isThreeOfAKing := ThreeOfAKind(hand)
	if isThreeOfAKing {
		used[index] = true
		return fichas, mult * 3, gold, used
	}
	return fichas, mult, gold, used
}

func Photograph(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	for _, card := range hand.Cards {
		if grade(card) >= 11 { // J=11, Q=12, K=13
			used[index] = true
			return fichas, mult * 2, gold, used
		}
	}
	return fichas, mult, gold, used
}

func Petpet(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return fichas, mult + gold, gold, used
}

func EmptyJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	randomNumber := rand.Intn(50) + 1
	if randomNumber == 1 {
		used[index] = true
		return fichas + 25, mult + 200, gold, used
	}
	return fichas, mult, gold, used
}

func TwoFriendsJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	if fichas < 10 {
		diff := 10 - fichas
		return fichas, mult + diff, gold, used
	}

	return fichas - 10, mult + 10, gold, used
}

func HellCowboy(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	max := 0
	for _, card := range hand.Cards {
		if grade(card) > max {
			max = grade(card)
		}
	}
	return fichas, mult + max, gold, used
}

func LiriliLarila(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	for _, card := range hand.Cards {
		if grade(card) == 2 {
			mult += 2
		}
	}
	return fichas, mult * 2, gold, used
}

func BIRDIFICATION(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	var cardGrade int
	for _, card := range hand.Cards {
		cardGrade = grade(card)
		switch cardGrade {
		case 1, 4, 6, 7:
			used[index] = true
			fichas += 50
		default:
		}
	}
	return fichas, mult, gold, used
}

func ApplyJokers(hand Hand, js Jokers, initialFichas int, initialMult int, currentGold int) (int, int, int, []bool) {
	currentFichas, currentMult, currentGold := initialFichas, initialMult, currentGold
	used := make([]bool, len(js.Juglares)) // Jokers triggereados

	for i, jokerID := range js.Juglares {
		if jokerID == 0 {
			continue
		}

		if jokerFunc, exists := jokerTable[jokerID]; exists {
			// Apply joker and update state
			currentFichas, currentMult, currentGold, used = jokerFunc(hand, currentFichas, currentMult, currentGold, used, i)
		} else {
			fmt.Printf("Warning: Unknown joker ID — what is %d?\n", jokerID)
		}
	}

	return currentFichas, currentMult, currentGold, used
}
