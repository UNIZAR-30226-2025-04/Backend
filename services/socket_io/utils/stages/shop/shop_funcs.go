package shop

import (
	game_constants "Nogler/constants/game"
	"Nogler/models/redis"
	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"golang.org/x/exp/rand"
)

const ( // Only used here, i think its good to see it here
	minFixedPacks        = 2
	maxFixedPacks        = 4
	minModifiers         = 1
	maxModifiers         = 3
	jokersCount          = 3
	fixedPackPrefix      = "fixed_pack_"
	fixedModifierPrefix  = "fixed_mod_"
	rerollableItemPrefix = "rerollable_item_"
)

func InitializeShop(lobbyID string, roundNumber int) (*redis.LobbyShop, error) {
	baseSeed := GenerateSeed(lobbyID, "shop", roundNumber)
	rng := rand.New(rand.NewSource(baseSeed))

	shop := &redis.LobbyShop{
		Rerolls:         0,
		FixedPacks:      generateFixedPacks(rng),
		FixedModifiers:  generateFixedModifiers(rng),
		RerollableItems: generateRerollableItems(rng, jokersCount),
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
			ID:       fmt.Sprintf("%s%d", fixedPackPrefix, i),
			Type:     game_constants.PACK_TYPE,
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

	// Calculate total weight
	totalWeight := 0
	for _, modifier := range poker.ModifierWeights {
		totalWeight += modifier.Weight
	}

	for i := range modifiers {
		// Generate weighted random modifier ID
		randomWeight := rng.Intn(totalWeight)
		modifierID := 1 // Default to 1 in case something goes wrong

		for _, modifier := range poker.ModifierWeights {
			if randomWeight < modifier.Weight {
				modifierID = modifier.ID
				break
			}
			randomWeight -= modifier.Weight
		}

		modifiers[i] = redis.ShopItem{
			ID:         fmt.Sprintf("%s%d", fixedModifierPrefix, i),
			Type:       game_constants.MODIFIER_TYPE,
			Price:      50 + rng.Intn(50),
			ModifierId: modifierID,
		}
	}
	return modifiers
}

// TODO: how tf do we do this
func RerollShopItems(redisClient *redis_services.RedisClient, lobbyID string) error {
	// do stuff
	return nil
}

func generateRerollableItems(rng *rand.Rand, count int) []redis.ShopItem {
	// NOTE: only jokers are rerrollable items
	rerollableItems := make([]redis.ShopItem, count)
	var jokers []poker.Jokers
	jokers = generateJokers(rng, count)

	for i := range rerollableItems {
		rerollableItems[i] = redis.ShopItem{
			ID:      fmt.Sprintf("%s%d", rerollableItemPrefix, i),
			Type:    game_constants.JOKER_TYPE,
			Price:   50 + rng.Intn(50),
			JokerId: jokers[i].Juglares[0], // Assuming we want the first joker
			// NOTE: only needed for packs
			// PackSeed: rng.Int63(),
		}
	}
	return rerollableItems
}

// TODO: Add a function that applies the probabilities of the groups (joker card modifier)

// TODO: Add a function that calculates the probabilities of a given item in a group to appear

func GetOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem) (*redis.PackContents, error) {
	// Unique key per pack state
	packKey := fmt.Sprintf("lobby:%s:round:%d:reroll:%d:pack:%s",
		lobby.Id, lobby.MaxRounds, lobby.ShopState.Rerolls, item.ID)

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

func FindShopItem(lobby redis.GameLobby, itemID int) (redis.ShopItem, bool) {
	// Iterate over the shop items in the lobby
	for _, item := range lobby.ShopState.FixedPacks {
		if item.ID == fmt.Sprintf("fixed_pack_%d", itemID) {
			return item, true
		}
	}

	for _, item := range lobby.ShopState.FixedModifiers {
		if item.ID == fmt.Sprintf("fixed_mod_%d", itemID) {
			return item, true
		}
	}

	for _, item := range lobby.ShopState.RerollableItems {
		if item.ID == fmt.Sprintf("rerollable_item_%d", itemID) {
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

// PurchaseJoker processes the purchase of a joker by a player
func PurchaseJoker(redisClient *redis_services.RedisClient, player *redis.InGamePlayer,
	item redis.ShopItem, clientPrice int) (bool, *redis.InGamePlayer, error) {

	if err := ValidatePurchase(item, game_constants.JOKER_TYPE, clientPrice, player); err != nil {
		return false, nil, err
	}

	// Get the current jokers from player's inventory
	var currentJokers poker.Jokers
	if player.CurrentJokers != nil && len(player.CurrentJokers) > 0 {
		if err := json.Unmarshal(player.CurrentJokers, &currentJokers); err != nil {
			return false, nil, fmt.Errorf("error parsing player's jokers: %v", err)
		}
	} else {
		// Initialize empty jokers array if none exists
		currentJokers = poker.Jokers{
			Juglares: []int{},
		}
	}

	// Add the joker to player's collection
	jokerID := item.JokerId
	currentJokers.Juglares = append(currentJokers.Juglares, jokerID)

	// Deduct the price from player's money
	player.PlayersMoney -= item.Price

	// Update player's joker inventory
	updatedJokersJSON, err := json.Marshal(currentJokers)
	if err != nil {
		return false, nil, fmt.Errorf("error updating jokers: %v", err)
	}
	player.CurrentJokers = updatedJokersJSON

	return true, player, nil
}

// PurchaseVoucher processes the purchase of a modifier/voucher by a player
func PurchaseVoucher(redisClient *redis_services.RedisClient, player *redis.InGamePlayer,
	item redis.ShopItem, clientPrice int) (bool, *redis.InGamePlayer, error) {

	if err := ValidatePurchase(item, game_constants.MODIFIER_TYPE, clientPrice, player); err != nil {
		return false, nil, err
	}

	// Get the current modifiers from player's inventory
	var currentModifiers poker.Modifiers
	if player.Modifiers != nil && len(player.Modifiers) > 0 {
		if err := json.Unmarshal(player.Modifiers, &currentModifiers); err != nil {
			return false, nil, fmt.Errorf("error parsing player's modifiers: %v", err)
		}
	} else {
		// Initialize empty modifiers array if none exists
		currentModifiers = poker.Modifiers{
			Modificadores: []poker.Modifier{},
		}
	}

	// Use the ModifierId field directly instead of parsing it from the item ID
	modifierID := item.ModifierId

	// Add the new modifier to player's collection
	newModifier := poker.Modifier{
		Value: modifierID,
		// TODO: should be removed, we're not supposed to use it for now
		LeftUses: -1, // Set to -1 if it doesn't expire until the end of the game, or set a specific value
	}
	currentModifiers.Modificadores = append(currentModifiers.Modificadores, newModifier)

	// Deduct the price from player's money
	player.PlayersMoney -= item.Price

	// Update player's modifier inventory
	updatedModifiersJSON, err := json.Marshal(currentModifiers)
	if err != nil {
		return false, nil, fmt.Errorf("error updating modifiers: %v", err)
	}
	player.Modifiers = updatedModifiersJSON

	return true, player, nil
}

// ValidatePurchase performs common validation for item purchases
func ValidatePurchase(item redis.ShopItem, expectedType string, clientPrice int, player *redis.InGamePlayer) error {
	// Verify the item type
	if item.Type != expectedType {
		return fmt.Errorf("item is not a %s", expectedType)
	}

	// Verify that client-provided price matches the server's price
	if clientPrice != item.Price {
		return fmt.Errorf("price mismatch: expected %d, got %d", item.Price, clientPrice)
	}

	// Check if player has enough money
	if player.PlayersMoney < item.Price {
		return fmt.Errorf("insufficient funds: need %d, have %d", item.Price, player.PlayersMoney)
	}

	return nil
}
