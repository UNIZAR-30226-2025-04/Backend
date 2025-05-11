package poker

import (
	"fmt"
	"log"
	"math"
	"sort"

	"golang.org/x/exp/rand"
)

type Jokers struct {
	Juglares []int
}

type JokerFunc func(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool)

const ExpSellPriceDenominator = 9

var jokerTable = map[int]JokerFunc{
	// Common
	1: SolidSevenJoker,
	2: PoorJoker,
	3: Petpet,
	4: AverageSizeMichel,
	5: HellCowboy,
	6: CarbSponge,
	7: TwoFriendsJoker,
	8: BIRDIFICATION,

	// Uncommon
	9:  Photograph,
	10: EmptyJoker,
	11: LiriliLarila,
	12: Rustyahh,
	13: damnapril,
	14: crowave,
	15: bicicleta,
	16: salebalatrito,
	17: diego_joker,
	18: itssoover,

	// Rare
	19: paris,
	20: nasus,
	21: sombrilla,
}

// 5

func SolidSevenJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return fichas + 7, mult + 7, gold, used
}

func AverageSizeMichel(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	randomNumber := rand.Intn(15) + 1
	if randomNumber == 1 {
		// Manage destroy the joker
	}
	return fichas, mult + 15, gold, used
}

func PoorJoker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	used[index] = true
	return fichas, mult, gold + 4, used
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
	// None should be negative
	total := 14 + rand.Intn(6)     // 14-19
	maxDelta := min(total, fichas) // Previene fichas negativas
	delta := rand.Intn(maxDelta + 1)

	return fichas + delta, mult + (total - delta), gold, used
}

func itssoover(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {

	// +10 de oro si solo se juega 1 carta (mano de tamaño 1)
	if len(hand.Cards) == 1 {
		used[index] = true
		gold += 10
	}

	return fichas, mult, gold, used
}

func paris(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	// +3 mult por cada pareja de cartas del mismo palo
	suitCount := make(map[string]int)
	for _, card := range hand.Cards {
		suitCount[card.Suit]++
	}
	pairs := 0
	for _, count := range suitCount {
		pairs += count / 2
		used[index] = true

	}
	return fichas, mult + (3 * pairs), gold, used
}

func diego_joker(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {

	// Solo activa si se juegan EXACTAMENTE 3 cartas
	if len(hand.Cards) == 3 {
		used[index] = true

		mult *= 4 // Multiplicador x4
	}

	return fichas, mult, gold, used
}

func bicicleta(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {

	// Contar cuántos 2 hay en la mano
	countTwos := 0
	for _, card := range hand.Cards {
		if grade(card) == 2 { // Asume que grade() devuelve 2 para cartas de valor 2
			used[index] = true

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

	return fichas, max(mult*gold, 1), gold, used
}

func sombrilla(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {

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
		used[index] = true

	}

	return fichas, mult, gold, used
}

func salebalatrito(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {

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
		used[index] = true

		fichas += 50
	}

	return fichas, mult, gold, used
}

func kaefece(hand Hand, fichas int, mult int, gold int, used []bool, index int) (int, int, int, []bool) {
	// Contar cartas negras
	darkCards := 0
	for _, card := range hand.Cards {
		if card.Suit == "Spades" || card.Suit == "Clubs" {
			used[index] = true

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

	// Count red cards (hearts/diamonds)
	redCards := 0
	for _, card := range hand.Cards {
		if card.Suit == "Hearts" || card.Suit == "Diamonds" {

			used[index] = true
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

func ApplyJokers(hand Hand, js Jokers, initialFichas int, initialMult int, currentGold int, username string) (int, int, int, []bool) {
	currentFichas, currentMult, currentGold := initialFichas, initialMult, currentGold
	used := make([]bool, len(js.Juglares)) // Jokers triggereados

	for i, jokerID := range js.Juglares {
		if jokerID == 0 {
			continue
		}

		if jokerFunc, exists := jokerTable[jokerID]; exists {
			// Apply joker and update state
			currentFichas, currentMult, currentGold, used = jokerFunc(hand, currentFichas, currentMult, currentGold, used, i)
			log.Println("[JOKER-APPLIED] User: ", username, "Joker", jokerID, "Fichas:", currentFichas, "Mult:", currentMult, "Gold:", currentGold)
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

var (
	// rarity probabilities
	RarityProbabilities = map[string]int{
		"Common":   70, // 70% total chance
		"Uncommon": 25, // 25% total chance
		"Rare":     5,  // 5% total chance
	}

	// ID ranges for each rarity tier
	RarityRanges = map[string][]int{
		"Common":   {1, 8},
		"Uncommon": {9, 18},
		"Rare":     {19, 21},
	}
)

func GenerateJokers(rng *rand.Rand, numJokers int) []Jokers {
	// Group jokers by their rarity based on ID ranges
	rarityGroups := make(map[string][]int)
	for id := range jokerTable {
		var found bool
		for rarity, bounds := range RarityRanges {
			if len(bounds) != 2 {
				continue
			}
			if id >= bounds[0] && id <= bounds[1] {
				rarityGroups[rarity] = append(rarityGroups[rarity], id)
				found = true
				break
			}
		}
		if !found {
			// Skip IDs outside defined ranges
			continue
		}
	}

	// Filter out rarities with no available jokers
	var availableRarities []string
	availableWeights := make(map[string]int)
	totalWeight := 0

	for rarity, weight := range RarityProbabilities {
		if len(rarityGroups[rarity]) > 0 {
			availableRarities = append(availableRarities, rarity)
			availableWeights[rarity] = weight
			totalWeight += weight
		}
	}
	sort.Strings(availableRarities) // For deterministic selection

	if totalWeight == 0 {
		panic("no available jokers in any rarity tier")
	}

	// Generate jokers
	jokers := make([]Jokers, numJokers)
	for i := range jokers {
		// Select rarity tier
		randomWeight := rng.Intn(totalWeight)
		selectedRarity := ""

		for _, rarity := range availableRarities {
			if randomWeight < availableWeights[rarity] {
				selectedRarity = rarity
				break
			}
			randomWeight -= availableWeights[rarity]
		}

		// Select specific joker from chosen rarity group
		group := rarityGroups[selectedRarity]
		jokerID := group[rng.Intn(len(group))]

		jokers[i] = Jokers{Juglares: []int{jokerID}}
	}

	return jokers
}

func GetJokerPrice(jokerID int) int {
	// Common jokers (IDs 1-8)
	if jokerID >= 1 && jokerID <= 8 {
		return 2
	}
	// Uncommon jokers (IDs 9-18)
	if jokerID >= 9 && jokerID <= 18 {
		return 4
	}
	// Rare jokers (IDs 19-21)
	if jokerID >= 19 && jokerID <= 21 {
		return 6
	}
	return -104 // IDK I LIKE THE NUMBER, SHOULD NOT HAPPEN
}
