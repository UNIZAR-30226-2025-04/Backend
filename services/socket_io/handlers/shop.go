package handlers

import (
	redis "Nogler/models/redis"

	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	socketio_utils "Nogler/services/socket_io/utils"
	"log"

	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

// Handler that will be called.
func HandlerOpenPack(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {

		log.Printf("OpenPack iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 2 {
			log.Printf("[HAND-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el pack a abrir o la lobby ID"})
			return
		}

		// Handle pack ID (JavaScript numbers come as float64 says depsik)
		packIDFloat, ok := args[0].(float64)
		if !ok {
			client.Emit("error", gin.H{"error": "El pack ID debe ser un número"})
			return
		}
		packID := int(packIDFloat) // Convert to int
		lobbyID := args[1].(string)

		log.Printf("[INFO] Obteniendo información del lobby ID: %s para usuario: %s", lobbyID, username)

		// Check if lobby exists
		var lobby redis.GameLobby
		if err := db.Where("id = ?", lobbyID).First(&lobby).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				client.Emit("error", gin.H{"error": "Lobby not found"})
			} else {
				client.Emit("error", gin.H{"error": "Database error"})
			}
			return
		}

		item, exists := findShopItem(lobby, packID)
		if !exists || item.Type != "pack" {
			client.Emit("invalid_pack")
			return
		}

		contents, err := getOrGeneratePackContents(redisClient, &lobby, item)
		if err != nil {
			client.Emit("pack_generation_failed")
			return
		}

		client.Emit("pack_opened", gin.H{
			"cards":  contents.Cards,
			"jokers": contents.Jokers,
		})
	}
}

const ( // Only used here, i think its good to see it here
	minFixedPacks = 2
	maxFixedPacks = 4
	minModifiers  = 1
	maxModifiers  = 3
)

func InitializeShop(lobbyID string, roundNumber int) (*redis.LobbyShop, error) {
	baseSeed := socketio_utils.GenerateSeed(lobbyID, "shop", roundNumber)
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
			Price:    socketio_utils.CalculatePackPrice(3), // 3 should really be the number of items
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

func getOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem) (*redis.PackContents, error) {
	// Unique key per pack state
	packKey := fmt.Sprintf("lobby:%s:round:%d:reroll:%d:pack:%s",
		lobby.Id, lobby.NumberOfRounds, lobby.ShopState.Rerolls, item.ID)

	// Try to get existing pack contents from Redis
	contents, err := rc.GetPackContents(packKey)
	if err != nil {
		return nil, err
	}
	if contents != nil {
		return contents, nil
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
	for _, joker := range socketio_utils.JokerWeights {
		totalWeight += joker.Weight
	}

	// Generate jokers based on probabilities
	jokers := make([]poker.Jokers, numJokers)
	for i := 0; i < numJokers; i++ {
		randomWeight := rng.Intn(totalWeight)
		for _, joker := range socketio_utils.JokerWeights {
			if randomWeight < joker.Weight {
				jokers[i] = poker.Jokers{Juglares: []int{joker.ID}}
				break
			}
			randomWeight -= joker.Weight
		}
	}

	return jokers
}

func findShopItem(lobby redis.GameLobby, packID int) (redis.ShopItem, bool) {
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
