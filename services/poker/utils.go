package poker

import (
	"encoding/json"
)

// Marshal the deck for Redis storage
func (d *Deck) ToJSON() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"total_cards":  d.TotalCards,
		"played_cards": d.PlayedCards,
	})
	return data
}

// Unmarshal from Redis data
func DeckFromJSON(data json.RawMessage) (*Deck, error) {
	var result struct {
		Total  []Card `json:"total_cards"`
		Played []Card `json:"played_cards"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &Deck{
		TotalCards:  result.Total,
		PlayedCards: result.Played,
	}, nil
}

func InitializePlayerDeck() json.RawMessage {
	deck := NewStandardDeck() // Creates deck with TotalCards
	deck.Shuffle()
	return deck.ToJSON()
}

// Generate all combinations of hands of a given size from a deck of cards
func GenerateHands(hand []Card, handSize int) [][]Card {
	combinations := [][]Card{}
	combination := make([]Card, handSize)
	var generate func(int, int)
	generate = func(start, depth int) {
		if depth == handSize {
			combinationCopy := make([]Card, handSize)
			copy(combinationCopy, combination)
			combinations = append(combinations, combinationCopy)
			return
		}
		for i := start; i < len(hand); i++ {
			combination[depth] = hand[i]
			generate(i+1, depth+1)
		}
	}
	generate(0, 0)
	return combinations
}
