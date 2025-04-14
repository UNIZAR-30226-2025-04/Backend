package shop

import (
	"Nogler/models/redis"
	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	"fmt"
	"hash/fnv"
	"time"

	"golang.org/x/exp/rand"
)

const ( // Only used here, i think its good to see it here
	minFixedPacks = 2
	maxFixedPacks = 4
	minModifiers  = 1
	maxModifiers  = 3
)

func InitializeShop(lobbyID string, roundNumber int) (*redis.LobbyShop, error) {
	baseSeed := GenerateSeed(lobbyID, "shop", roundNumber)
	rng := rand.New(rand.NewSource(baseSeed))

	shop := &redis.LobbyShop{
		Rerolls:         0,
		FixedPacks:      generateFixedPacks(rng),
		FixedModifiers:  generateFixedModifiers(rng),
		RerollableItems: generateRerollableItems(rng, 4),
	}

	return shop, nil
}

func generateFixedPacks(rng *rand.Rand) []redis.ShopItem {
	// Wrong, think a feasable number of packs generated per shop
	// Could be managed by seing maxmoney, rounds maxmoneyplayer can reroll, and calc
	count := minFixedPacks + rng.Intn(maxFixedPacks-minFixedPacks+1)
	packs := make([]redis.ShopItem, count)

	for i := range packs {
		seed := rng.Int63()
		packs[i] = redis.ShopItem{
			ID:       fmt.Sprintf("fixed_pack_%d", i),
			Type:     "pack",
			Price:    CalculatePackPrice(3), // 3 should really be the number of items
			PackSeed: seed,
		}
	}
	return packs
}

func generateFixedModifiers(rng *rand.Rand) []redis.ShopItem {
	// Same count problem as fixedpacks
	count := minModifiers + rng.Intn(maxModifiers-minModifiers+1)
	modifiers := make([]redis.ShopItem, count)

	for i := range modifiers {
		modifiers[i] = redis.ShopItem{
			ID:       fmt.Sprintf("fixed_mod_%d", i),
			Type:     "modifier",
			Price:    50 + rng.Intn(50),
			PackSeed: rng.Int63(),
		}
	}
	return modifiers
}

func RerollShopItems(redisClient *redis_services.RedisClient, lobbyID string) error {
	// do stuff
	return nil
}

func generateRerollableItems(rng *rand.Rand, count int) []redis.ShopItem {
	// do stuff
	return nil
}

// Add a function that applies the probabilities of the groups (joker card modifier)

// Add a function that calculates the probabilities of a given item in a group to appear

func GetOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem) (*redis.PackContents, error) {
	// Unique key per pack state
	packKey := fmt.Sprintf("lobby:%s:round:%d:reroll:%d:pack:%s",
		lobby.Id, lobby.NumberOfRounds, lobby.ShopState.Rerolls, item.ID)

	// Try to get existing pack contents
	existing, err := rc.GetPackContents(packKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Generate new contents if not found
	newContents := generatePackContents(uint64(item.PackSeed))

	if err := rc.SetPackContents(packKey, newContents, 24*time.Hour); err != nil {
		return nil, err
	}

	return &newContents, nil
}

func generatePackContents(seed uint64) redis.PackContents {
	rng := rand.New(rand.NewSource(seed))
	numItems := minFixedPacks + rng.Intn(maxFixedPacks-minFixedPacks+1)

	// Determine number of jokers (can be 0)
	numJokers := rng.Intn(numItems + 1) // Ensure jokers + cards = numItems

	numCards := numItems - numJokers

	return redis.PackContents{
		Cards:  generateCards(rng, numCards),
		Jokers: generateJokers(rng, numJokers),
	}
}

// Predefined slices for ranks and suits, we dont want to recalculate each time. might not be the best modularity but makes sense here
var ranks = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
var suits = []string{"h", "d", "c", "s"}

func generateCards(rng *rand.Rand, numCards int) []poker.Card {
	cards := make([]poker.Card, numCards)

	// Generate random cards
	for i := 0; i < numCards; i++ {
		rank := ranks[rng.Intn(len(ranks))]
		suit := suits[rng.Intn(len(suits))]
		cards[i] = poker.Card{Rank: rank, Suit: suit}
	}

	return cards
}

func generateJokers(rng *rand.Rand, numJokers int) []poker.Jokers {
	// Calculate total weight
	totalWeight := 0
	for _, joker := range JokerWeights {
		totalWeight += joker.Weight
	}

	// Generate jokers based on probabilities
	jokers := make([]poker.Jokers, numJokers)
	for i := 0; i < numJokers; i++ {
		randomWeight := rng.Intn(totalWeight)
		for _, joker := range JokerWeights {
			if randomWeight < joker.Weight {
				jokers[i] = poker.Jokers{Juglares: []int{joker.ID}}
				break
			}
			randomWeight -= joker.Weight
		}
	}

	return jokers
}

func FindShopItem(lobby redis.GameLobby, packID int) (redis.ShopItem, bool) {
	// Iterate over the shop items in the lobby
	for _, item := range lobby.ShopState.FixedPacks {
		if item.ID == fmt.Sprintf("fixed_pack_%d", packID) {
			return item, true
		}
	}

	for _, item := range lobby.ShopState.FixedModifiers {
		if item.ID == fmt.Sprintf("fixed_mod_%d", packID) {
			return item, true
		}
	}

	for _, item := range lobby.ShopState.RerollableItems {
		if item.ID == fmt.Sprintf("rerollable_item_%d", packID) {
			return item, true
		}
	}

	// If no match is found, return false
	return redis.ShopItem{}, false
}

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
