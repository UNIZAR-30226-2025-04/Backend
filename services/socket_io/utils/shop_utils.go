package socketio_utils

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
)

func GenerateSeed(parts ...interface{}) uint64 {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprint(parts...)))
	return uint64(h.Sum64())
}

func CalculatePackPrice(numItems int) int {
	return numItems + 1
}

// Change this OBVIOUSLY GPT GENERATED for a real one
func RandomModifierType(rng *rand.Rand) string {
	return "modifier yeahhhhh"
}

// Change weights
var JokerWeights = []struct {
	ID     int
	Weight int
}{
	{1, 10}, // SolidSevenJoker: 10% chance
	{2, 20}, // PoorJoker: 20% chance
	{3, 15}, // BotardoJoker: 15% chance
	{4, 10}, // AverageSizeMichel: 10% chance
	{5, 5},  // HellCowboy: 5% chance
	{6, 10}, // CarbSponge: 10% chance
	{7, 10}, // Photograph: 10% chance
	{8, 10}, // Petpet: 10% chance
	{9, 5},  // EmptyJoker: 5% chance
	{10, 5}, // TwoFriendsJoker: 5% chance
}
