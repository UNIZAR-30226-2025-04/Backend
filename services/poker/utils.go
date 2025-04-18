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

func generatePack() {

}
