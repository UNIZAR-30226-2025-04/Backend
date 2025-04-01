package poker

import (
	"math/rand"
	"sort"
	"strconv"
)

type Hand struct {
	Cards  []Card `json:"cards"`
	Jokers Jokers `json:"jokers"`
	Gold   int    `json:"gold"`
}

type Deck struct {
	TotalCards  []Card `json:"total_cards"`
	PlayedCards []Card `json:"played_cards"`
}

// Define a Card struct with a Rank and suit
type Card struct {
	Rank string
	Suit string
}

// Cards A, 2, 3, 4, 5, 6, 7, 8, 9, 10, J, Q, K
// Suit s (spades), c (clubs), d (diamonds), h (hearts)

type Multiplier struct {
	First  int
	Second int
}

// Temporal table of fichas - mult for testing. will be fetched eventually
var TypeMap = map[string]Multiplier{
	"RoyalFlush":    {65, 50},
	"StraightFlush": {50, 40},
	"FlushFive":     {35, 25},
	"FlushHouse":    {32, 22},
	"FiveOfAKind":   {30, 20},
	"FourOfAKind":   {25, 15},
	"FullHouse":     {20, 12},
	"Flush":         {15, 8},
	"Straight":      {12, 5},
	"ThreeOfAKind":  {10, 4},
	"TwoPair":       {8, 3},
	"Pair":          {4, 2},
	"HighCard":      {1, 1},
}

func NewStandardDeck() *Deck {
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	suits := []string{"h", "d", "c", "s"}

	total := make([]Card, 0, 52)
	for _, suit := range suits {
		for _, rank := range ranks {
			total = append(total, Card{Rank: rank, Suit: suit})
		}
	}

	return &Deck{
		TotalCards:  total,
		PlayedCards: make([]Card, 0),
	}
}

func (d *Deck) AddCards(newCards []Card) {
	d.TotalCards = append(d.TotalCards, newCards...)
}

func (d *Deck) RemoveCards(toRemove []Card) {
	newTotal := make([]Card, 0, len(d.TotalCards))

	for _, card := range d.TotalCards {
		keep := true
		for _, removed := range toRemove {
			if card.Rank == removed.Rank && card.Suit == removed.Suit {
				keep = false
				break
			}
		}
		if keep {
			newTotal = append(newTotal, card)
		}
	}

	d.TotalCards = newTotal
}

func (d *Deck) MarkAsPlayed(cards []Card) {
	d.PlayedCards = append(d.PlayedCards, cards...)
}

func (d *Deck) Draw(n int) []Card {
	if len(d.TotalCards) < n {
		d.reshufflePlayed()
	}

	if n > len(d.TotalCards) {
		n = len(d.TotalCards)
	}

	drawn := d.TotalCards[:n]
	d.TotalCards = d.TotalCards[n:]

	return drawn
}

// Shuffle randomizes the deck using Fisher-Yates algorithm
func (d *Deck) Shuffle() {

	// If we have played cards, combine them back first
	if len(d.PlayedCards) > 0 {
		d.TotalCards = append(d.TotalCards, d.PlayedCards...)
		d.PlayedCards = []Card{} // Clear played cards
	}

	// Fisher-Yates shuffle on TotalCards (deepseek lo dice yo escucho)
	for i := len(d.TotalCards) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		d.TotalCards[i], d.TotalCards[j] = d.TotalCards[j], d.TotalCards[i]
	}
}

// Necesario para si p ejemplo me quedan 3 cartas por drawear y juego 5 pues reshufleo
func (d *Deck) reshufflePlayed() {

	// Mezclar played cards
	rand.Shuffle(len(d.PlayedCards), func(i, j int) {
		d.PlayedCards[i], d.PlayedCards[j] = d.PlayedCards[j], d.PlayedCards[i]
	})

	// Añadir al final del mazo
	d.TotalCards = append(d.TotalCards, d.PlayedCards...)
	d.PlayedCards = make([]Card, 0)
}

func grade(c1 Card) int {
	switch c1.Rank {
	case "K":
		return 13
	case "Q":
		return 12
	case "J":
		return 11
	case "A":
		return 1
	default:
		rank, _ := strconv.Atoi(c1.Rank)
		return rank
	}
}

// Get the final CHIPS a fixed card will score when played as part of a determined hand
// Face cards are worth 10 chips, numerated cards are worth the rank chips, and aces are worth
// 11 chips unless played as the start of an A-2-3-4-5, where it is worth 1 chip
func PointsPerCard(hand Hand, c Card) int {
	var value int
	switch c.Rank {
	case "K", "Q", "J":
		value = 10
	case "A":
		value = 11
	default:
		value, _ = strconv.Atoi(c.Rank)
	}
	return value
}

func sortCards(h *Hand) {
	sort.Slice(h.Cards, func(i, j int) bool {
		return grade(h.Cards[i]) < grade(h.Cards[j])
	})
}

func countRanks(h Hand) map[int]int {
	counts := make(map[int]int)
	for _, c := range h.Cards {
		counts[grade(c)]++
	}
	return counts
}

// -------------------------------------------------------------------------------------------------

func Pair(h Hand) bool {
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

func TwoPair(h Hand) bool {
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

func ThreeOfAKind(h Hand) bool {
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

func FullHouse(h Hand) bool {
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

func Flush(h Hand) bool {
	suit := h.Cards[0].Suit
	for _, c := range h.Cards[1:] {
		if c.Suit != suit {
			return false
		}
	}
	return true
}

func Straight(h Hand) bool {
	// Create sorted copy
	tmp := Hand{Cards: make([]Card, len(h.Cards))}
	copy(tmp.Cards, h.Cards)
	sortCards(&tmp)

	grades := make([]int, len(tmp.Cards))
	for i, c := range tmp.Cards {
		grades[i] = grade(c)
	}

	// Check normal straight
	for i := 0; i < len(grades)-1; i++ {
		if grades[i+1]-grades[i] != 1 {
			// Check Ace-low straight (A-2-3-4-5)
			if grades[len(grades)-1] == 14 { // Ace high
				grades = append([]int{1}, grades[:len(grades)-1]...)
				sort.Ints(grades)
				for i := 0; i < len(grades)-1; i++ {
					if grades[i+1]-grades[i] != 1 {
						return false
					}
				}
				return true
			}
			return false
		}
	}
	return true
}

func StraightFlush(h Hand) bool {
	return Straight(h) && Flush(h)
}

func FiveOfAKind(h Hand) bool {
	return grade(h.Cards[0]) == grade(h.Cards[1]) && grade(h.Cards[1]) == grade(h.Cards[2]) && grade(h.Cards[2]) == grade(h.Cards[3]) && grade(h.Cards[3]) == grade(h.Cards[4])
}

func FourOfAKind(h Hand) bool {
	counts := countRanks(h)
	for _, v := range counts {
		if v == 4 {
			return true
		}
	}
	return false
}

func RoyalFlush(h Hand) bool {
	if !StraightFlush(h) {
		return false
	}

	// Check for 10-J-Q-K-A
	required := map[int]bool{10: true, 11: true, 12: true, 13: true, 14: true}
	grades := make(map[int]bool)
	for _, c := range h.Cards {
		grades[grade(c)] = true
	}

	for rank := range required {
		if !grades[rank] {
			return false
		}
	}
	return true
}

func FlushHouse(h Hand) bool {
	return Flush(h) && FullHouse(h)
}

// Flush + todas iguales
func FlushFive(h Hand) bool {
	return FiveOfAKind(h) && Flush(h)
}

// EL ÚLTIMO INT QUE DEVUELVE SIGNIFICA LO SIGUIENTE:
//
// Es un valor que se asocia a un tipo de mano. Esta fijado de la siguiente forma:
// RoyalFlush = 1
// StraightFlush = 2
// FiveOfAKind = 3
// FlushFive = 4
// FlushHouse = 5
// FourOfAKind = 6
// FullHouse = 7
// Flush = 8
// Straight = 9
// ThreeOfAKind = 10
// TwoPair = 11
// Pair = 12
// HighCard = 13

func BestHand(h Hand) (int, int, int) {
	// Make a copy to avoid modifying original
	tmp := Hand{Cards: make([]Card, len(h.Cards))}
	copy(tmp.Cards, h.Cards)
	sortCards(&tmp)

	// Check for the strongest hand first and return as soon as we find one

	switch {
	case RoyalFlush(tmp):
		return TypeMap["RoyalFlush"].First, TypeMap["RoyalFlush"].Second, 1
	case StraightFlush(tmp):
		return TypeMap["StraightFlush"].First, TypeMap["StraightFlush"].Second, 2
	case FiveOfAKind(tmp):
		return TypeMap["FiveOfAKind"].First, TypeMap["FiveOfAKind"].Second, 3
	case FlushFive(tmp):
		return TypeMap["FlushFive"].First, TypeMap["FlushFive"].Second, 4
	case FlushHouse(tmp):
		return TypeMap["FluushHouse"].First, TypeMap["FlushHouse"].Second, 5
	case FourOfAKind(tmp):
		return TypeMap["FourOfAKind"].First, TypeMap["FourOfAKind"].Second, 6
	case FullHouse(tmp):
		return TypeMap["FullHouse"].First, TypeMap["FullHouse"].Second, 7
	case Flush(tmp):
		return TypeMap["Flush"].First, TypeMap["Flush"].Second, 8
	case Straight(tmp):
		return TypeMap["Straight"].First, TypeMap["Straight"].Second, 9
	case ThreeOfAKind(tmp):
		return TypeMap["ThreeOfAKind"].First, TypeMap["ThreeOfAKind"].Second, 10
	case TwoPair(tmp):
		return TypeMap["TwoPair"].First, TypeMap["TwoPair"].Second, 11
	case Pair(tmp):
		return TypeMap["Pair"].First, TypeMap["Pair"].Second, 12
	default:
		return TypeMap["HighCard"].First, TypeMap["HighCard"].Second, 13
	}
}
