package poker

import (
	"fmt"
	"math"
	"math/rand"
)

type Jokers struct {
	Juglares []int
}

type JokerFunc func(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool)

// TODO, use it?
const MaxJokers = 10
const ExpSellPriceDenominator = 9

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
	13: Rustyahh,
	14: damnapril,
	15: itssoover,
	16: paris,
	17: diego_joker,
	18: bicicleta,
	19: nasus,
	20: sombrilla,
	21: salebalatrito,
	22: kaefece,
	23: crowave,
}

// 5

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

func Rustyahh(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return fichas, mult * 2, 0, used
}

func damnapril(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	// Asegura que fichas no sea negativo
	total := 14 + rand.Intn(6)     // 14-19
	maxDelta := min(total, fichas) // Previene fichas negativas
	delta := rand.Intn(maxDelta + 1)

	return fichas + delta, mult + (total - delta), gold, used
}

func itssoover(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true

	// +10 de oro si solo se juega 1 carta (mano de tamaño 1)
	if len(hand.Cards) == 1 {
		gold += 10
	}

	return fichas, mult, gold, used
}

func paris(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	// +3 mult por cada pareja de cartas del mismo palo
	suitCount := make(map[string]int)
	for _, card := range hand.Cards {
		suitCount[card.Suit]++
	}
	pairs := 0
	for _, count := range suitCount {
		pairs += count / 2
	}
	return fichas, mult + (3 * pairs), gold, used
}

func diego_joker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true

	// Solo activa si se juegan EXACTAMENTE 3 cartas
	if len(hand.Cards) == 3 {
		mult *= 4 // Multiplicador x4
	}

	return fichas, mult, gold, used
}

func bicicleta(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true

	// Contar cuántos 2 hay en la mano
	countTwos := 0
	for _, card := range hand.Cards {
		if grade(card) == 2 { // Asume que grade() devuelve 2 para cartas de valor 2
			countTwos++
		}
	}

	// Aplicar bonus por cada 2
	mult += countTwos * 2
	fichas += countTwos * 20

	return fichas, mult, gold, used
}

func nasus(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return fichas, mult * gold, gold, used
}

func sombrilla(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true

	// Check for face cards (J=11, Q=12, K=13, A=1/14)
	hasFaceCard := false
	for _, card := range hand.Cards {
		val := grade(card)
		if val >= 11 || val == 1 {
			hasFaceCard = true
			break
		}
	}

	// Add +20 Mult if no face cards played
	if !hasFaceCard {
		mult += 20
	}

	return fichas, mult, gold, used
}

func salebalatrito(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true

	// Contar cuántas veces aparece cada valor de carta
	valueCounts := make(map[int]int)
	for _, card := range hand.Cards {
		valueCounts[grade(card)]++
	}

	// Verificar si hay al menos un trío (3 cartas del mismo valor)
	hasTrio := false
	for _, count := range valueCounts {
		if count >= 3 {
			hasTrio = true
			break
		}
	}

	// +50 fichas si hay trío
	if hasTrio {
		fichas += 50
	}

	return fichas, mult, gold, used
}

func kaefece(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	// Contar cartas negras
	darkCards := 0
	for _, card := range hand.Cards {
		if card.Suit == "Spades" || card.Suit == "Clubs" {
			darkCards++
		}
	}

	// Efecto 1 (+5 Mult por carta negra)
	bonus := 5
	mult += darkCards * bonus

	if darkCards >= 4 {
		fichas += 50
		mult *= 2
	}

	return fichas, mult * 2, gold, used
}

func crowave(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true

	// Count red cards (hearts/diamonds)
	redCards := 0
	for _, card := range hand.Cards {
		if card.Suit == "Hearts" || card.Suit == "Diamonds" {
			redCards++
		}
	}

	// 90%: Add to mult (original effect)
	// 10%: Add to fichas instead
	if rand.Intn(100) < 90 {
		mult += redCards * 3 // +3 mult per red card
	} else {
		fichas += redCards * 5 // Alternative: +5 fichas per red card
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

// Returns a joker's sell price based on its ID.
// The max price for a joker with ID 23 will be e^(23/9) = e^2.55 ~= 12
func CalculateJokerSellPrice(jokerID int) int {
	return int(math.Exp(float64(jokerID / ExpSellPriceDenominator)))
}
