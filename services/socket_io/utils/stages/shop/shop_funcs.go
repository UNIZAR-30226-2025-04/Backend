package shop

import (
	"Nogler/models/redis"
	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	"fmt"
	"hash/fnv"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"golang.org/x/exp/rand"
)

const ( // Only used here, i think its good to see it here
	numFixedPacks = 3
	numModifiers  = 2
	numJokers     = 3
	minFixedPacks = 2
	maxFixedPacks = 4
)

// The Shop has (according to frontend)
// 3 rellolable jokers
// 2 Non rerollable consumables
// 2 NOn rerollable packs

func InitializeShop(lobbyID string, roundNumber int) (*redis.LobbyShop, error) {
	baseSeed := GenerateSeed(lobbyID, "shop", roundNumber)
	rng := rand.New(rand.NewSource(baseSeed))

	shop := &redis.LobbyShop{
		MaxRerrol:        0,
		FixedPacks:       generateFixedPacks(rng),
		FixedModifiers:   generateFixedModifiers(rng),
		RerollableJokers: generateInitialJokers(rng, 3),
	}

	return shop, nil
}

func generateFixedPacks(rng *rand.Rand) []redis.ShopItem {

	packs := make([]redis.ShopItem, numFixedPacks)

	for i := range packs {
		packs[i] = redis.ShopItem{
			ID:       fmt.Sprintf("%d", i),
			Type:     "pack",
			Price:    CalculatePackPrice(3), // 3 should really be the number of items
			PackSeed: rng.Int63(),
		}
	}
	return packs
}

// TODO IMPORTANT change so that it  generates valid modifiers was done before modifiers were a thing, take a look at generateinitialjokers
func generateFixedModifiers(rng *rand.Rand) []redis.ShopItem {
	// Same count problem as fixedpacks
	modifiers := make([]redis.ShopItem, numModifiers)

	for i := range modifiers {
		modifiers[i] = redis.ShopItem{
			ID:       fmt.Sprintf("%d", i),
			Type:     "modifier",
			Price:    2, // See how we calculate this price, probably another CalculateModPrice like the one above
			PackSeed: rng.Int63(),
		}
	}
	return modifiers
}

func generateInitialJokers(rng *rand.Rand, numJokers int) []redis.ShopItem {

	totalWeight := 120 // CHANGE TO THE TOTAL WEIGHT + MOVE TO CONSTANTS

	jokers := make([]redis.ShopItem, numJokers)
	for i := range jokers {
		randomWeight := rng.Intn(totalWeight)
		remainingWeight := randomWeight

		for _, joker := range JokerWeights {
			if remainingWeight < joker.Weight {
				jokers[i] = redis.ShopItem{
					ID:       fmt.Sprintf("%d", joker.ID), // Unique ID
					Type:     "joker",
					Price:    calculateJokerPrice(joker.ID),
					PackSeed: rng.Int63(),
				}
				break
			}
			remainingWeight -= joker.Weight
		}
	}
	return jokers
}

// Good way of calculating price, needs ordering of jokers by rarity, do same with probabilities instead of the "weight" thing
func calculateJokerPrice(jokerID int) int {
	// Simple tiered pricing based on weight ranges. Change this logically, just a placeholder for now but works for testiun
	switch {
	case jokerID <= 3: // Common jokers
		return 3
	case jokerID <= 7: // Uncommon
		return 5
	default: // Rare
		return 8
	}
}

// pasar el socket
func RerollShopItems(redisClient *redis_services.RedisClient, lobbyID string, username string, client *socket.Socket, rng *rand.Rand) error {

	// Recuperar del in_game_player el numero de rerolls
	// Recuperar de la lobby el max rerolls
	// Comparar

	player, err := redisClient.GetInGamePlayer(username)
	if err != nil {
		log.Printf("[BLIND-ERROR] Error getting player data: %v", err)
		client.Emit("error", gin.H{"error": "Error getting player data"})
		return fmt.Errorf("error getting player data: %v", err)
	}
	gamePlayerRerrols := player.ShopRerolls + 1
	player.ShopRerolls = gamePlayerRerrols

	if err := redisClient.SaveInGamePlayer(player); err != nil {
		log.Printf("[BLIND-ERROR] Error saving player data: %v", err)
		client.Emit("error", gin.H{"error": "Error saving player data"})
		return fmt.Errorf("error saving player data: %v", err)
	}

	lobby, err := redisClient.GetGameLobby(lobbyID)
	if err != nil {
		log.Printf("[BLIND-ERROR] Error getting game lobby: %v", err)
		client.Emit("error", gin.H{"error": "Error getting game lobby"})
		return fmt.Errorf("error getting game lobby: %v", err)
	}
	lobbyMaxRerrols := lobby.ShopState.MaxReroll

	if gamePlayerRerrols > lobbyMaxRerrols {

		newSeed := rng.Uint64()                          // Use the RNG's current seed
		newSeed += uint64(lobby.NumberOfRounds) << 32    // Mix in round number
		newSeed += uint64(lobby.ShopState.MaxReroll + 1) // Mix in reroll count
		rng := rand.New(rand.NewSource(newSeed))

		generateJokers(rng, 3)
		redisClient.SaveRerollJokers()

	} else {
		jokers, err := redisClient.GetJokersFromRound(lobbyID)
		if err != nil {
			log.Printf("[BLIND-ERROR] Error getting jokers from this rounds shop: %v", err)
			client.Emit("error", gin.H{"error": "Error getting jokers from this rounds shop"})
			return fmt.Errorf("error getting jokers: %v", err)
		}

		client.Emit("pack_opened", gin.H{
			"jokers": jokers,
		})
	}
	return nil

}

// Add a function that applies the probabilities of the groups (joker card modifier)

// Add a function that calculates the probabilities of a given item in a group to appear

func GetOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem, username string) (*redis.PackContents, error) {
	// Unique key per pack state
	packKey := fmt.Sprintf("lobby:%s:round:%d:reroll:%d:pack:%s",
		lobby.Id, lobby.NumberOfRounds, lobby.ShopState.MaxRerrol, item.ID)

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
