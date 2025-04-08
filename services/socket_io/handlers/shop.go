package handlers

import (
	redis "Nogler/models/redis"
	redis_services "Nogler/services/redis"
	socketio_utils "Nogler/services/socket_io/utils"

	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"golang.org/x/exp/rand"
	"gorm.io/gorm"
)

func generateShopSeed(lobbyID string, roundNumber, rerollCount int) int64 {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%s-%d-%d", lobbyID, roundNumber, rerollCount)))
	return int64(h.Sum64())
}

const ( // Only used here, i think its good to see it here
	minFixedPacks = 2
	maxFixedPacks = 3
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
	return redisClient.Watch(func(tx *redis.Tx) error {
		lobby, err := tx.GetLobby(lobbyID)
		if err != nil {
			return err
		}

		// Only regenerate rerollable items
		newRerollCount := lobby.ShopState.RerollCount + 1
		seed := generateSeed(lobbyID, "reroll", lobby.ShopState.RoundNumber, newRerollCount)
		rng := rand.New(rand.NewSource(seed))

		lobby.ShopState.RerollableItems = generateRerollableItems(rng, 4)
		lobby.ShopState.RerollCount = newRerollCount

		return tx.SaveLobby(lobby)
	})
}

func generateRerollableItems(rng *rand.Rand, count int) []redis.ShopItem {
	items := make([]redis.ShopItem, count)
	for i := range items {
		itemType := weightedRandomType(rng)
		items[i] = redis.ShopItem{
			ID:       fmt.Sprintf("reroll_item_%d", i),
			Type:     itemType,
			Price:    getPriceByType(rng, itemType),
			Seed:     rng.Int63(),
			Metadata: generateItemMetadata(rng, itemType),
		}
	}
	return items
}

func weightedRandomType(rng *rand.Rand) string {
	// Example weights: 40% card, 30% modifier, 20% joker, 10% pack
	roll := rng.Float32()
	switch {
	case roll < 0.4:
		return "card"
	case roll < 0.7:
		return "modifier"
	case roll < 0.9:
		return "joker"
	default:
		return "pack"
	}
}

func HandlerOpenPack(redisClient *redis_services.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {
		lobbyID, itemID := validateArgs(args)

		lobby, err := redisClient.GetLobby(lobbyID)
		if err != nil {
			client.EmitError("lobby_not_found")
			return
		}

		item, exists := findShopItem(lobby, itemID)
		if !exists || item.Type != "pack" {
			client.EmitError("invalid_pack")
			return
		}

		contents, err := getOrGeneratePackContents(redisClient, lobby, item)
		if err != nil {
			client.EmitError("pack_generation_failed")
			return
		}

		if err := saveUserItems(db, username, contents); err != nil {
			client.EmitError("inventory_update_failed")
			return
		}

		client.Emit("pack_opened", gin.H{
			"cards":  contents.Cards,
			"jokers": contents.Jokers,
		})
	}
}

func getOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem) (*PackContents, error) {
	// Unique key per pack state
	packKey := fmt.Sprintf("lobby:%s:round:%d:reroll:%d:pack:%s",
		lobby.Id, lobby.ShopState.RoundNumber, lobby.ShopState.RerollCount, item.ID)

	var contents PackContents
	if err := rc.Get(packKey, &contents); err == nil {
		return &contents, nil
	}

	// Generate new contents
	contents = generatePackContents(item.Seed, item.Metadata)
	if err := rc.Set(packKey, contents, 24*time.Hour); err != nil {
		return nil, err
	}

	return &contents, nil
}

func generatePackContents(seed int64, metadata string) PackContents {
	rng := rand.New(rand.NewSource(seed))
	var params struct {
		Cards  int `json:"cards"`
		Jokers int `json:"jokers"`
	}
	json.Unmarshal([]byte(metadata), &params)

	return PackContents{
		Cards:  generateCards(rng, params.Cards),
		Jokers: generateJokers(rng, params.Jokers),
	}
}
