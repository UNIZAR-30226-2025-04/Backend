package poker

import (
	"sort"
	"strconv"
)

// Define a Card struct with a Rank and suit
type Card struct {
	Rank string
	Suit string
}

// Cards A, 2, 3, 4, 5, 6, 7, 8, 9, 10, J, Q, K
// Suit s (spades), c (clubs), d (diamonds), h (hearts)

const (
	FlushFive     = "Flush five"
	FlushHouse    = "Flush house"
	FiveOfAKind   = "Fife of a kind"
	RoyalFlush    = "Royal Flush"
	StraightFlush = "Straight Flush"
	FourOfAKind   = "Four of a Kind"
	FullHouse     = "Full House"
	Flush         = "Flush"
	Straight      = "Straight"
	ThreeOfAKind  = "Three of a Kind"
	TwoPair       = "Two Pair"
	Pair          = "Pair"
	HighCard      = "High Card"
)

type Hand struct {
	Cards []Card
}

func grade(c1 Card) int {
	cr := c1.Rank
	if cr != "K" && cr != "Q" && cr != "J" && cr != "A" {
		rank, _ := strconv.Atoi(cr)
		return rank
	} else if cr == "K" {
		return 13
	} else if cr == "Q" {
		return 12
	} else if cr == "J" {
		return 11
	} else {
		return 1
	}
}

func compareCards(c1 Card, c2 Card) int {
	if grade(c1) > grade(c2) {
		return 1
	} else if grade(c1) < grade(c2) {
		return -1
	} else {
		return 0
	}
}

func sortCards(h Hand) {
	sort.Slice(h.Cards, func(i, j int) bool {
		return grade(h.Cards[i]) > grade(h.Cards[j])
	})
}

func isPair(h Hand) bool {
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}
	for _, count := range cardCount {
		if count == 2 {
			return true
		}
	}
	return false
}

func twoPair(h Hand) bool {
	// Create a map to count the occurrences of each rank
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	// Count how many pairs we have
	pairCount := 0
	for _, count := range cardCount {
		if count == 2 {
			pairCount++
		}
	}

	// If there are exactly two pairs, return true
	return pairCount == 2
}

func threeOfAKind(h Hand) bool {
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}
	for _, count := range cardCount {
		if count == 2 {
			return true
		}
	}
	return false
}

func fullHouse(h Hand) bool {
	// Create a map to count the occurrences of each rank
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	// Flags to track if we found three of a kind and a pair
	hasThree := false
	hasTwo := false

	// Check the counts to identify a three of a kind and a pair
	for _, count := range cardCount {
		if count == 3 {
			hasThree = true
		} else if count == 2 {
			hasTwo = true
		}
	}

	// A full house requires exactly one three of a kind and one pair
	return hasThree && hasTwo
}

func flush(h Hand) bool {
	return h.Cards[0].Suit == h.Cards[1].Suit && h.Cards[0].Suit == h.Cards[2].Suit && h.Cards[0].Suit == h.Cards[3].Suit && h.Cards[0].Suit == h.Cards[4].Suit
}

func straight(h Hand) bool {
	return grade(h.Cards[0]) == grade(h.Cards[1])-1 && grade(h.Cards[1]) == grade(h.Cards[2])-1 && grade(h.Cards[2]) == grade(h.Cards[3])-1 && grade(h.Cards[3]) == grade(h.Cards[4])-1
}

func straightFlush(h Hand) bool {
	return straight(h) && flush(h)
}

func fiveOfAKind(h Hand) bool {
	return grade(h.Cards[0]) == grade(h.Cards[1]) && grade(h.Cards[1]) == grade(h.Cards[2]) && grade(h.Cards[2]) == grade(h.Cards[3]) && grade(h.Cards[3]) == grade(h.Cards[4])
}

func fourOfAKind(h Hand) bool {
	return grade(h.Cards[0]) == grade(h.Cards[1]) && grade(h.Cards[1]) == grade(h.Cards[2]) && grade(h.Cards[2]) == grade(h.Cards[3])
}

func royalFlush(h Hand) bool {
	return straightFlush(h) && h.Cards[0].Rank == "10"
}

func flushHouse(h Hand) bool {
	return flush(h) && fullHouse(h)
}

// Flush + todas iguales
func flushFive(h Hand) bool {
	return fiveOfAKind(h) && flush(h)
}

func BestHand(h Hand) string {
	// Sort the hand by rank to help with evaluating hands like straight or full house
	sortCards(h)

	// Check for the strongest hand first and return as soon as we find one

	// TODO!!!!!!
	if royalFlush(h) {
		return RoyalFlush
	}
	if straightFlush(h) {
		return StraightFlush
	}
	if fiveOfAKind(h) {
		return FiveOfAKind
	}
	if flushFive(h) {
		return FlushFive
	}
	if fourOfAKind(h) {
		return FourOfAKind
	}
	if fullHouse(h) {
		return FullHouse
	}
	if flush(h) {
		return Flush
	}
	if straight(h) {
		return Straight
	}
	if threeOfAKind(h) {
		return ThreeOfAKind
	}
	if twoPair(h) {
		return TwoPair
	}
	if isPair(h) {
		return Pair
	}
	// If no other hand matches, return High Card
	return HighCard
}
