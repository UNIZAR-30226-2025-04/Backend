package poker

import "fmt"

type Jokers struct {
	Juglares []int
}

type JokerFunc func(hand Hand, fichas int, mult int) (int, int)

var jokerTable = map[int]JokerFunc{
	1: SolidSevenJoker,
	2: DoubleFichasJoker,
	3: DoubleMultJoker,
	4: PlusTenIfPair,
}

func SolidSevenJoker(hand Hand, fichas int, mult int) (int, int) {
	return fichas + 7, mult + 7
}

func PlusTenIfPair(hand Hand, fichas int, mult int) (int, int) {
	if TwoPair(hand) {
		return fichas, mult + 10
	}
	return fichas, mult
}

func DoubleFichasJoker(hand Hand, fichas int, mult int) (int, int) {
	return fichas * 2, mult
}

func DoubleMultJoker(hand Hand, fichas int, mult int) (int, int) {
	return fichas, mult * 2
}

func ApplyJokers(hand Hand, js Jokers, initialFichas int, initialMult int) (int, int) {
	currentFichas, currentMult := initialFichas, initialMult

	for _, jokerID := range js.Juglares {
		if jokerID == 0 {
			continue // Skip zero values
		}

		if jokerFunc, exists := jokerTable[jokerID]; exists {
			currentFichas, currentMult = jokerFunc(hand, currentFichas, currentMult)
		} else {
			fmt.Printf("Warning: Unknown joker ID idk what you are doing, what is %d\n", jokerID)
		}
	}

	return currentFichas, currentMult
}
