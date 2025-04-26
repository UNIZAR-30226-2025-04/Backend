package shop

import (
	game_constants "Nogler/constants/game"
	"Nogler/models/redis"
	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	redis_utils "Nogler/services/redis/utils"
	"encoding/json"
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
	jokersCount   = 3
)

func InitializeShop(lobbyID string, roundNumber int) (*redis.LobbyShop, error) {
	baseSeed := GenerateSeed(lobbyID, "shop", roundNumber)
	rng := rand.New(rand.NewSource(baseSeed))
	// NEW: unique ID for each shop item
	nextUniqueId := 1

	shop := &redis.LobbyShop{
		Rerolls:        0,
		FixedPacks:     generateFixedPacks(rng, &nextUniqueId),
		FixedModifiers: generateFixedModifiers(rng, &nextUniqueId),
		// NOTE: fixed number of rerollable items
		RerollableItems: generateRerollableItems(rng, jokersCount, &nextUniqueId),
	}

	return shop, nil
}

func generateFixedPacks(rng *rand.Rand, nextUniqueId *int) []redis.ShopItem {
	// Wrong, think a feasable number of packs generated per shop
	// Could be managed by seing maxmoney, rounds maxmoneyplayer can reroll, and calc
	count := minFixedPacks + rng.Intn(maxFixedPacks-minFixedPacks+1)
	packs := make([]redis.ShopItem, count)

	for i := range packs {
		seed := rng.Int63()
		packs[i] = redis.ShopItem{
			ID:       *nextUniqueId,
			Type:     game_constants.PACK_TYPE,
			Price:    CalculatePackPrice(3), // 3 should really be the number of items
			PackSeed: seed,
		}
		*nextUniqueId++
	}
	return packs
}

func generateFixedModifiers(rng *rand.Rand, nextUniqueId *int) []redis.ShopItem {
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
			ID:         *nextUniqueId,
			Type:       game_constants.MODIFIER_TYPE,
			Price:      2, // 50 + rng.Intn(50), TODO, CHANGE, Emilliano estaba pobre
			ModifierId: modifierID,
		}

		*nextUniqueId++
	}
	return modifiers
}

// TODO: how tf do we do this
func RerollShopItems(redisClient *redis_services.RedisClient, lobbyID string) error {
	// do stuff
	return nil
}

func generateRerollableItems(rng *rand.Rand, count int, nextUniqueId *int) []redis.ShopItem {
	// NOTE: only jokers are rerrollable items
	rerollableItems := make([]redis.ShopItem, count)
	var jokers []poker.Jokers
	jokers = generateJokers(rng, count)

	for i := range rerollableItems {
		rerollableItems[i] = redis.ShopItem{
			ID:      *nextUniqueId,
			Type:    game_constants.JOKER_TYPE,
			Price:   2,                     // 50 + rng.Intn(50), TODO, CHANGE, Emilliano estaba pobre
			JokerId: jokers[i].Juglares[0], // Assuming we want the first joker
			// NOTE: only needed for packs
			// PackSeed: rng.Int63(),
		}

		*nextUniqueId++
	}
	return rerollableItems
}

// TODO: Add a function that applies the probabilities of the groups (joker card modifier)

// TODO: Add a function that calculates the probabilities of a given item in a group to appear

func GetOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem) (*redis.PackContents, error) {
	// Unique key per pack state

	packKey := redis_utils.FormatPackKey(lobby.Id, lobby.MaxRounds, lobby.ShopState.Rerolls, item.ID)
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
		if item.ID == itemID {
			return item, true
		}
	}

	for _, item := range lobby.ShopState.FixedModifiers {
		if item.ID == itemID {
			return item, true
		}
	}

	for _, item := range lobby.ShopState.RerollableItems {
		if item.ID == itemID {
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

// SellJoker processes the sale of a joker by a player
// It returns the updated player state, sell price, and any error
func SellJoker(player *redis.InGamePlayer, jokerID int) (updatedPlayer *redis.InGamePlayer, sellPrice int, err error) {
	// Parse current jokers
	var currentJokers poker.Jokers
	if player.CurrentJokers == nil || len(player.CurrentJokers) == 0 {
		return nil, 0, fmt.Errorf("no jokers in inventory")
	}

	if err := json.Unmarshal(player.CurrentJokers, &currentJokers); err != nil {
		return nil, 0, fmt.Errorf("error parsing jokers: %v", err)
	}

	// Check if player has the joker
	foundIndex := -1
	for i, id := range currentJokers.Juglares {
		if id == jokerID {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return nil, 0, fmt.Errorf("joker not found in inventory")
	}

	// Calculate sell price
	sellPrice = poker.CalculateJokerSellPrice(jokerID)

	// Remove joker from inventory (by replacing it with the last element and then truncating)
	lastIdx := len(currentJokers.Juglares) - 1
	currentJokers.Juglares[foundIndex] = currentJokers.Juglares[lastIdx]
	currentJokers.Juglares = currentJokers.Juglares[:lastIdx]

	// Update player's joker inventory
	updatedJokersJSON, err := json.Marshal(currentJokers)
	if err != nil {
		return nil, 0, fmt.Errorf("error updating jokers: %v", err)
	}
	player.CurrentJokers = updatedJokersJSON

	// Add sell price to player's money
	player.PlayersMoney += sellPrice

	return player, sellPrice, nil
}

// ProcessPackSelection validates and processes a player's selection from a purchased pack
func ProcessPackSelection(redisClient *redis_services.RedisClient, lobby *redis.GameLobby,
	player *redis.InGamePlayer, itemID int, selectedCardMap map[string]interface{},
	selectedJokerID int) (*redis.InGamePlayer, error) {

	// Convert selected card map to poker.Card
	rankInterface, hasRank := selectedCardMap["Rank"]
	suitInterface, hasSuit := selectedCardMap["Suit"]
	enhancementInterface, hasEnhancement := selectedCardMap["Enhancement"]

	if !hasRank || !hasSuit {
		return nil, fmt.Errorf("selected card is missing rank or suit")
	}

	rank, rankOk := rankInterface.(string)
	suit, suitOk := suitInterface.(string)

	if !rankOk || !suitOk {
		return nil, fmt.Errorf("card rank and suit must be strings")
	}

	// Create the card with Enhancement if available
	selectedCard := poker.Card{
		Rank: rank,
		Suit: suit,
	}

	// Add enhancement if present
	if hasEnhancement {
		if enhancementValue, ok := enhancementInterface.(float64); ok {
			selectedCard.Enhancement = int(enhancementValue)
		}
	}

	// Get pack contents
	packKey := redis_utils.FormatPackKey(lobby.Id, lobby.MaxRounds, lobby.ShopState.Rerolls, itemID)
	packContents, err := redisClient.GetPackContents(packKey)
	if err != nil || packContents == nil {
		return nil, fmt.Errorf("pack contents not found for item ID %d", itemID)
	}

	// Verify the selected card exists in the pack
	cardFound := false
	for _, card := range packContents.Cards {
		// TODO, check if we should really check enhancement
		if card.Rank == selectedCard.Rank && card.Suit == selectedCard.Suit && card.Enhancement == selectedCard.Enhancement {
			cardFound = true
			break
		}
	}
	if !cardFound {
		return nil, fmt.Errorf("the selected card is not in the pack")
	}

	// Verify the selected joker exists in the pack
	jokerFound := false
	for _, jokerGroup := range packContents.Jokers {
		for _, jokerID := range jokerGroup.Juglares {
			if jokerID == selectedJokerID {
				jokerFound = true
				break
			}
		}
		if jokerFound {
			break
		}
	}
	if !jokerFound {
		return nil, fmt.Errorf("the selected joker is not in the pack")
	}

	// Store selected card in PurchasedPackCards instead of CurrentDeck
	var purchasedCards []poker.Card
	if player.PurchasedPackCards != nil && len(player.PurchasedPackCards) > 0 {
		if err := json.Unmarshal(player.PurchasedPackCards, &purchasedCards); err != nil {
			return nil, fmt.Errorf("error parsing player's purchased cards: %v", err)
		}
	} else {
		purchasedCards = []poker.Card{}
	}
	purchasedCards = append(purchasedCards, selectedCard)
	updatedPurchasedCardsJSON, err := json.Marshal(purchasedCards)
	if err != nil {
		return nil, fmt.Errorf("error updating purchased cards: %v", err)
	}
	player.PurchasedPackCards = updatedPurchasedCardsJSON

	// Add selected joker to player's jokers
	var currentJokers poker.Jokers
	if player.CurrentJokers != nil && len(player.CurrentJokers) > 0 {
		if err := json.Unmarshal(player.CurrentJokers, &currentJokers); err != nil {
			return nil, fmt.Errorf("error parsing player's jokers: %v", err)
		}
	} else {
		currentJokers = poker.Jokers{
			Juglares: []int{},
		}
	}
	currentJokers.Juglares = append(currentJokers.Juglares, selectedJokerID)
	updatedJokersJSON, err := json.Marshal(currentJokers)
	if err != nil {
		return nil, fmt.Errorf("error updating jokers: %v", err)
	}
	player.CurrentJokers = updatedJokersJSON

	// Reset LastPurchasedPackItemId to prevent reuse
	player.LastPurchasedPackItemId = -1

	return player, nil
}
