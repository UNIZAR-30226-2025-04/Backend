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
	Rank        string
	Suit        string
	Enhancement int // 0 nada 1 = +30 chips; 2 = +4 mult
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
	"FiveOfAKind":   {30, 20},
	"FlushHouse":    {32, 22},
	"FlushFive":     {35, 25},
	"FourOfAKind":   {25, 15},
	"FullHouse":     {20, 12},
	"Flush":         {15, 8},
	"Straight":      {12, 5},
	"ThreeOfAKind":  {10, 4},
	"TwoPair":       {8, 3},
	"Pair":          {4, 2},
	"HighCard":      {1, 1},
}

var RankMap = map[string]bool{
	"A": true, "2": true, "3": true, "4": true, "5": true,
	"6": true, "7": true, "8": true, "9": true, "10": true,
	"J": true, "Q": true, "K": true,
}

var SuitMap = map[string]bool{
	"h": true, "d": true, "c": true, "s": true,
}

func NewStandardDeck() *Deck {
	total := make([]Card, 0, 52)

	// Iterate over SuitMap and RankMap keys
	for suit := range SuitMap {
		for rank := range RankMap {
			total = append(total, Card{Rank: rank, Suit: suit, Enhancement: 0})
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
		return 14
	default:
		rank, _ := strconv.Atoi(c1.Rank)
		return rank
	}
}

// Get the final CHIPS a fixed card will score when played as part of a determined hand
// Face cards are worth 10 chips, numerated cards are worth the rank chips, and aces are worth
func PointsPerCard(c Card) int {
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

func SortCards(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		return grade(cards[i]) < grade(cards[j])
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

func Pair(h Hand) ([]Card, bool) {
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	var scoringCards []Card
	for rank, count := range cardCount {
		if count == 2 {
			// Find the cards that match this rank
			for _, card := range h.Cards {
				if card.Rank == rank {
					scoringCards = append(scoringCards, card)
				}
			}
			return scoringCards, true
		}
	}
	return nil, false
}

func TwoPair(h Hand) ([]Card, bool) {
	// Create a map to count the occurrences of each rank
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	// Count how many pairs we have
	var scoringCards []Card
	pairCount := 0
	for rank, count := range cardCount {
		if count == 2 {
			pairCount++
			// Find the cards that match this rank
			for _, card := range h.Cards {
				if card.Rank == rank {
					scoringCards = append(scoringCards, card)
				}
			}
		}
	}

	// If there are exactly two pairs, return true
	if pairCount == 2 {
		return scoringCards, true
	}
	return nil, false
}

func ThreeOfAKind(h Hand) ([]Card, bool) {
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	var scoringCards []Card
	for rank, count := range cardCount {
		if count == 3 {
			// Find the cards that match this rank
			for _, card := range h.Cards {
				if card.Rank == rank {
					scoringCards = append(scoringCards, card)
				}
			}
			return scoringCards, true
		}
	}
	return nil, false
}

func FullHouse(h Hand) ([]Card, bool) {
	// Create a map to count the occurrences of each rank
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	var threeCards []Card
	var twoCards []Card

	// Check the counts to identify a three of a kind and a pair
	for rank, count := range cardCount {
		if count == 3 {
			// Find the cards that match this rank
			for _, card := range h.Cards {
				if card.Rank == rank {
					threeCards = append(threeCards, card)
				}
			}
		} else if count == 2 {
			// Find the cards that match this rank
			for _, card := range h.Cards {
				if card.Rank == rank {
					twoCards = append(twoCards, card)
				}
			}
		}
	}

	// A full house requires exactly one three of a kind and one pair
	if len(threeCards) == 3 && len(twoCards) == 2 {
		return append(threeCards, twoCards...), true
	}
	return nil, false
}

func Flush(h Hand) ([]Card, bool) {
	// Check if the hand has at least 5 cards
	if len(h.Cards) < 5 {
		return nil, false
	}

	suit := h.Cards[0].Suit
	var scoringCards []Card
	for _, c := range h.Cards {
		if c.Suit != suit {
			return nil, false
		}
		scoringCards = append(scoringCards, c)
	}
	return scoringCards, true
}

func Straight(h Hand) ([]Card, bool) {
	// Check if the hand has at least 5 cards
	if len(h.Cards) < 5 {
		return nil, false
	}

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
						return nil, false
					}
				}
				return tmp.Cards, true
			}
			return nil, false
		}
	}
	return tmp.Cards, true
}

func StraightFlush(h Hand) ([]Card, bool) {
	straightCards, isStraight := Straight(h)
	flushCards, isFlush := Flush(h)

	if isStraight && isFlush {
		// Filter the cards that are both in the straight and flush
		var scoringCards []Card
		for _, card := range straightCards {
			for _, flushCard := range flushCards {
				if card.Rank == flushCard.Rank && card.Suit == flushCard.Suit {
					scoringCards = append(scoringCards, card)
				}
			}
		}

		// If the number of cards in scoringCards is equal to the number of cards in straightCards, return them
		if len(scoringCards) == len(straightCards) {
			return scoringCards, true
		}
	}
	return nil, false
}

func FiveOfAKind(h Hand) ([]Card, bool) {
	cardCount := make(map[string]int)
	for _, card := range h.Cards {
		cardCount[card.Rank]++
	}

	// Check if any rank has 5 cards
	for rank, count := range cardCount {
		if count == 5 {
			var scoringCards []Card
			for _, card := range h.Cards {
				// Check if the card matches the rank
				if card.Rank == rank {
					scoringCards = append(scoringCards, card)
				}
			}
			return scoringCards, true
		}
	}
	return nil, false
}

func FourOfAKind(h Hand) ([]Card, bool) {
	counts := countRanks(h)

	for rank, v := range counts {
		if v == 4 {
			var scoringCards []Card
			for _, card := range h.Cards {
				if grade(card) == rank {
					scoringCards = append(scoringCards, card)
				}
			}
			return scoringCards, true
		}
	}
	return nil, false
}

func RoyalFlush(h Hand) ([]Card, bool) {
	straightFlushCards, isStraightFlush := StraightFlush(h)
	if !isStraightFlush {
		return nil, false
	}

	// Check for 10-J-Q-K-A
	required := map[int]bool{10: true, 11: true, 12: true, 13: true, 14: true}
	grades := make(map[int]bool)
	for _, c := range h.Cards {
		grades[grade(c)] = true
	}

	for rank := range required {
		if !grades[rank] {
			return nil, false
		}
	}
	return straightFlushCards, true
}

func FlushHouse(h Hand) ([]Card, bool) {
	flushCards, isFlush := Flush(h)
	fullHouseCards, isFullHouse := FullHouse(h)

	if isFlush && isFullHouse {
		return append(flushCards, fullHouseCards...), true
	}
	return nil, false
}

// Flush + todas iguales
func FlushFive(h Hand) ([]Card, bool) {
	fiveOfAKindCards, isFiveOfAKind := FiveOfAKind(h)
	flushCards, isFlush := Flush(h)

	if isFiveOfAKind && isFlush {
		return append(fiveOfAKindCards, flushCards...), true
	}
	return nil, false
}

// HighCard
func HighCard(h Hand) ([]Card, bool) {
	// Sort the cards in descending order
	sort.Slice(h.Cards, func(i, j int) bool {
		return grade(h.Cards[i]) > grade(h.Cards[j])
	})

	// Return the highest card
	return h.Cards[:1], true
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

func BestHand(h Hand) (int, int, int, []Card) {

	// NEW: handle the case with empty cards to avoid panics
	if len(h.Cards) <= 0 {
		return 0, 0, 0, nil
	}

	// Make a copy to avoid modifying original
	tmp := Hand{Cards: make([]Card, len(h.Cards))}
	copy(tmp.Cards, h.Cards)
	sortCards(&tmp)

	// Check for the strongest hand first and return as soon as we find one

	switch {
	case func(cards []Card, ok bool) bool { return ok }(RoyalFlush(tmp)):
		scoringCards, _ := RoyalFlush(tmp)
		return TypeMap["RoyalFlush"].First, TypeMap["RoyalFlush"].Second, 1, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(StraightFlush(tmp)):
		scoringCards, _ := StraightFlush(tmp)
		return TypeMap["StraightFlush"].First, TypeMap["StraightFlush"].Second, 2, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(FiveOfAKind(tmp)):
		scoringCards, _ := FiveOfAKind(tmp)
		return TypeMap["FiveOfAKind"].First, TypeMap["FiveOfAKind"].Second, 5, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(FlushHouse(tmp)):
		scoringCards, _ := FlushHouse(tmp)
		return TypeMap["FlushHouse"].First, TypeMap["FlushHouse"].Second, 4, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(FlushFive(tmp)):
		scoringCards, _ := FlushFive(tmp)
		return TypeMap["FlushFive"].First, TypeMap["FlushFive"].Second, 3, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(FourOfAKind(tmp)):
		scoringCards, _ := FourOfAKind(tmp)
		return TypeMap["FourOfAKind"].First, TypeMap["FourOfAKind"].Second, 6, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(FullHouse(tmp)):
		scoringCards, _ := FullHouse(tmp)
		return TypeMap["FullHouse"].First, TypeMap["FullHouse"].Second, 7, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(Flush(tmp)):
		scoringCards, _ := Flush(tmp)
		return TypeMap["Flush"].First, TypeMap["Flush"].Second, 8, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(Straight(tmp)):
		scoringCards, _ := Straight(tmp)
		return TypeMap["Straight"].First, TypeMap["Straight"].Second, 9, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(ThreeOfAKind(tmp)):
		scoringCards, _ := ThreeOfAKind(tmp)
		return TypeMap["ThreeOfAKind"].First, TypeMap["ThreeOfAKind"].Second, 10, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(TwoPair(tmp)):
		scoringCards, _ := TwoPair(tmp)
		return TypeMap["TwoPair"].First, TypeMap["TwoPair"].Second, 11, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(Pair(tmp)):
		scoringCards, _ := Pair(tmp)
		return TypeMap["Pair"].First, TypeMap["Pair"].Second, 12, scoringCards
	case func(cards []Card, ok bool) bool { return ok }(HighCard(tmp)):
		scoringCards, _ := HighCard(tmp)
		return TypeMap["HighCard"].First, TypeMap["HighCard"].Second, 13, scoringCards
	default:
		// If no hand is found, return 0
		return 0, 0, 0, nil
	}
}

func ApplyEnhancements(fichas int, mult int, cards []Card) (int, int) {
	for _, card := range cards {
		switch card.Enhancement {
		case 1:
			fichas += 30
		case 2:
			mult += 4
		default:
		}
	}
	return fichas, mult
}

func AddChipsPerCard(cards []Card) int {
	addition := 0
	for _, card := range cards {
		addition += PointsPerCard(card)
	}
	return addition
}
