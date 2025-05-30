package shop

import (
	game_constants "Nogler/constants/game"
	"Nogler/models/redis"
	"Nogler/services/poker"
	redis_services "Nogler/services/redis"
	redis_utils "Nogler/services/redis/utils"
	"Nogler/services/socket_io/utils/stages/play_round"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"time"

	"golang.org/x/exp/rand"
)

const ( // Only used here, i think its good to see it here
	minFixedPacks = 2
	maxFixedPacks = 4
	minModifiers  = 1
	maxModifiers  = 3
	jokersCount   = 3
	// Now, we only have 2 fixed packs and 2 fixed vouchers
	TOTAL_FIXED_PACKS    = 2
	TOTAL_FIXED_VOUCHERS = 2
)

func InitializeShop(lobbyID string, roundNumber int) (*redis.LobbyShop, error) {
	baseSeed := GenerateSeed(lobbyID, "shop", roundNumber)
	rng := rand.New(rand.NewSource(baseSeed))
	// NEW: unique ID for each shop item
	nextUniqueId := 1

	firstJokers := GenerateRerollableItems(rng, &nextUniqueId)
	shop := &redis.LobbyShop{
		Rerolls:        0,
		FixedPacks:     generateFixedPacks(rng, &nextUniqueId),
		FixedModifiers: generateFixedModifiers(rng, &nextUniqueId),
		// NOTE: fixed number of rerollable items
		Rerolled:     make([]redis.RerolledJokers, 0),
		RerollSeed:   GenerateSeed(lobbyID, "shop", roundNumber),
		NextUniqueId: nextUniqueId,
	}
	// Save first generated jokers as the rerolled 0
	shop.Rerolled = append(shop.Rerolled, firstJokers)

	return shop, nil
}

func generateFixedPacks(rng *rand.Rand, nextUniqueId *int) []redis.ShopItem {
	// Always generate exactly 2 packs
	packs := make([]redis.ShopItem, TOTAL_FIXED_PACKS)

	// Define possible pack types as integers
	packTypes := []int{
		game_constants.PACK_TYPE_CARDS,
		game_constants.PACK_TYPE_JOKERS,
		game_constants.PACK_TYPE_VOUCHERS,
	}

	// Randomly select 2 different pack types
	selectedTypes := make([]int, 2)

	// First pack type
	selectedTypes[0] = packTypes[rng.Intn(len(packTypes))]

	// Second pack type (ensure it's different from the first)
	var secondType int
	for {
		secondType = packTypes[rng.Intn(len(packTypes))]
		if secondType != selectedTypes[0] {
			selectedTypes[1] = secondType
			break
		}
	}

	// Generate each pack
	for i := 0; i < 2; i++ {
		seed := rng.Int63()
		packType := selectedTypes[i]

		// Set max selectable based on pack type
		var maxSelectable int
		switch packType {
		case game_constants.PACK_TYPE_CARDS:
			maxSelectable = 2
		case game_constants.PACK_TYPE_JOKERS:
			maxSelectable = 1
		case game_constants.PACK_TYPE_VOUCHERS:
			maxSelectable = 2
		default:
			maxSelectable = 1
		}

		packs[i] = redis.ShopItem{
			ID:            *nextUniqueId,
			Type:          game_constants.PACK_TYPE,
			Price:         calculatePackPrice(packType),
			PackSeed:      seed,
			PackType:      packType,
			MaxSelectable: maxSelectable,
		}

		*nextUniqueId++
	}

	return packs
}

// Update price calculation based on pack type
func calculatePackPrice(packType int) int {
	switch packType {
	case game_constants.PACK_TYPE_CARDS:
		return 3
	case game_constants.PACK_TYPE_JOKERS:
		return 4
	case game_constants.PACK_TYPE_VOUCHERS:
		return 3
	default:
		return 4
	}
}

func generateFixedModifiers(rng *rand.Rand, nextUniqueId *int) []redis.ShopItem {
	modifiers := make([]redis.ShopItem, TOTAL_FIXED_VOUCHERS)

	for i := range modifiers {
		// Simply generate a random number between 1 and 9
		modifierID := rng.Intn(9) + 1
		if modifierID == 0 {
			modifierID = 1
		}
		if modifierID == 2 || modifierID == 9 {
			modifierID = 3
		}

		modifiers[i] = redis.ShopItem{
			ID:         *nextUniqueId,
			Type:       game_constants.MODIFIER_TYPE,
			Price:      2, // Fixed price of 2
			ModifierId: modifierID,
		}

		*nextUniqueId++
	}
	return modifiers
}

func GenerateRerollableItems(rng *rand.Rand, nextUniqueId *int) redis.RerolledJokers {
	// NOTE: only jokers are rerrollable items
	rerollableItems := redis.RerolledJokers{}

	jokers := poker.GenerateJokers(rng, 3)

	for i := range 3 {
		rerollableItems.Jokers[i] = redis.ShopItem{
			ID:      *nextUniqueId, // tenemos en game_lobby el maxid, lo sacamos de ahi directamnete o lo pasamos a la función por param
			Type:    game_constants.JOKER_TYPE,
			Price:   poker.GetJokerPrice(jokers[i].Juglares[0]),
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

// TODO: would be nice to FLUSH all the redis content's after each game/round, WOULD BE VERY NICE

func GetOrGeneratePackContents(rc *redis_services.RedisClient, lobby *redis.GameLobby, item redis.ShopItem) (*redis.PackContents, error) {
	// Unique key per pack state
	packKey := redis_utils.FormatPackKey(lobby.Id, lobby.CurrentRound, item.ID)

	// Try to get existing pack contents
	existing, err := rc.GetPackContents(packKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Generate new contents if not found
	newContents := generatePackContents(uint64(item.PackSeed), item.PackType)

	log.Println("[GENERATE-PACK-CONTENTS] Pack type:", item.PackType)
	log.Println("[GENERATE-PACK-CONTENTS] Generated new pack contents:", newContents)

	if err := rc.SetPackContents(packKey, newContents, 24*time.Hour); err != nil {
		return nil, err
	}

	return &newContents, nil
}

func generatePackContents(seed uint64, packType int) redis.PackContents {
	rng := rand.New(rand.NewSource(seed))
	contents := redis.PackContents{
		Cards:    []poker.Card{},
		Jokers:   []poker.Jokers{},
		Vouchers: []poker.Modifier{},
	}

	log.Println("[GENERATE-PACK-CONTENTS] Pack type:", packType)

	switch packType {
	case game_constants.PACK_TYPE_CARDS:
		numCards := 4 + rng.Intn(3) // 4, 5, or 6 cards
		contents.Cards = generateCards(rng, numCards)

	case game_constants.PACK_TYPE_JOKERS:
		// Generate 3 jokers
		contents.Jokers = poker.GenerateJokers(rng, 3)

	case game_constants.PACK_TYPE_VOUCHERS:
		// Generate 3-4 vouchers (modifiers)
		numVouchers := 3 + rng.Intn(2) // 3 or 4
		contents.Vouchers = generatePackVouchers(rng, numVouchers)
	}

	log.Println("[GENERATE-PACK-CONTENTS] Pack contents:", contents)

	return contents
}

// Predefined slices for ranks and suits, we dont want to recalculate each time. might not be the best modularity but makes sense here
var ranks = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
var suits = []string{"h", "d", "c", "s"}
var enhancements = []int{0, 1, 2}

func generateCards(rng *rand.Rand, numCards int) []poker.Card {
	cards := make([]poker.Card, numCards)

	// Generate random cards
	for i := 0; i < numCards; i++ {
		rank := ranks[rng.Intn(len(ranks))]
		suit := suits[rng.Intn(len(suits))]
		enhancement := enhancements[rng.Intn(len(enhancements))]
		cards[i] = poker.Card{Rank: rank, Suit: suit, Enhancement: enhancement}
	}

	return cards
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

	// NEW: Check the jokers of the LATEST reroll
	total_rerolls_len := len(lobby.ShopState.Rerolled)
	log.Println("[FIND-SHOP-ITEM] Item ID:", itemID)
	log.Println("[FIND-SHOP-ITEM] Total rerolls length:", total_rerolls_len)

	if total_rerolls_len > 0 {
		log.Println("[FIND-SHOP-ITEM] Checking latest rerolled items: ", lobby.ShopState.Rerolled[total_rerolls_len-1])
		// Check the last rerolled items
		// NEW, CRITICAL: the player's current reroll might not be the last one, so we should check previous rerolls
		for _, reroll := range lobby.ShopState.Rerolled {
			for _, item := range reroll.Jokers {
				log.Println("[FIND-SHOP-ITEM] Checking item ID:", item.ID)
				if item.ID == itemID {
					return item, true
				}
			}
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

	// Check if adding another joker would exceed the maximum allowed
	if len(currentJokers.Juglares) >= game_constants.MaxJokersPerPlayer {
		return false, nil, fmt.Errorf("cannot have more than %d jokers", game_constants.MaxJokersPerPlayer)
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

	// NEW, KEY: set the corresponding purchased item IDs map entry to true
	play_round.SafelySetPlayerItemIDEntry(player, item)

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

	// NEW, KEY: set the corresponding purchased item IDs map entry to true
	play_round.SafelySetPlayerItemIDEntry(player, item)

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

	return player, sellPrice, nil
}

// ProcessPackSelection validates and processes a player's selection from a purchased pack
// Supports multiple pack types (cards, jokers, vouchers) and enforces MaxSelectable limit
func ProcessPackSelection(redisClient *redis_services.RedisClient, lobby *redis.GameLobby,
	player *redis.InGamePlayer, itemID int, selectionsMap map[string]interface{}, isCallFromBackend bool) (*redis.InGamePlayer, error) {

	// Get the shop item to determine pack type and MaxSelectable
	item, exists := FindShopItem(*lobby, itemID)
	if !exists {
		return nil, fmt.Errorf("pack not found in shop")
	}

	// Get pack contents
	packKey := redis_utils.FormatPackKey(lobby.Id, lobby.CurrentRound, itemID)
	packContents, err := redisClient.GetPackContents(packKey)
	if err != nil || packContents == nil {
		return nil, fmt.Errorf("pack contents not found for item ID %d", itemID)
	}

	// Parse selections based on pack type
	var selectedCards []poker.Card
	var selectedJokerIDs []int
	var selectedVoucherIDs []int
	totalSelected := 0

	// Parse selected cards if present
	if cardsInterface, hasCards := selectionsMap["selectedCards"]; hasCards {

		if isCallFromBackend {
			// Backend calls pass in []poker.Card directly
			backendCards, ok := cardsInterface.([]poker.Card)
			if !ok {
				return nil, fmt.Errorf("backend: selectedCards must be []poker.Card")
			}

			// Just use the cards directly
			selectedCards = backendCards
		} else {
			// Frontend calls pass JSON that becomes []interface{}
			frontendCards, ok := cardsInterface.([]interface{})
			if !ok {
				return nil, fmt.Errorf("frontend: selectedCards must be an array")
			}

			// Parse each card from the frontend format
			for _, cardInterface := range frontendCards {
				cardMap, ok := cardInterface.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("each card must be an object")
				}

				// Parse card fields
				rankInterface, hasRank := cardMap["Rank"]
				suitInterface, hasSuit := cardMap["Suit"]

				if !hasRank || !hasSuit {
					return nil, fmt.Errorf("card is missing rank or suit")
				}

				rank, rankOk := rankInterface.(string)
				suit, suitOk := suitInterface.(string)

				if !rankOk || !suitOk {
					return nil, fmt.Errorf("rank and suit must be strings")
				}

				// Create card
				card := poker.Card{
					Rank: rank,
					Suit: suit,
				}

				// Add enhancement if present
				if enhancementInterface, hasEnhancement := cardMap["Enhancement"]; hasEnhancement {
					if enhancementValue, ok := enhancementInterface.(float64); ok {
						card.Enhancement = int(enhancementValue)
					}
				}

				selectedCards = append(selectedCards, card)
			}
		}

		// Now selectedCards is populated correctly regardless of source
		totalSelected += len(selectedCards)
	}

	// Parse selected jokers if present
	if jokersInterface, hasJokers := selectionsMap["selectedJokers"]; hasJokers {

		if isCallFromBackend {
			// Backend calls pass in []int directly
			backendJokers, ok := jokersInterface.([]int)
			if !ok {
				return nil, fmt.Errorf("backend: selectedJokers must be []int")
			}

			// Just use the joker IDs directly
			selectedJokerIDs = backendJokers
		} else {
			// Frontend calls pass JSON that becomes []interface{}
			frontendJokers, ok := jokersInterface.([]interface{})
			if !ok {
				return nil, fmt.Errorf("frontend: selectedJokers must be an array")
			}

			// Parse each joker ID from the frontend format
			for _, jokerIDInterface := range frontendJokers {
				jokerIDFloat, ok := jokerIDInterface.(float64)
				if !ok {
					return nil, fmt.Errorf("each selected joker ID must be a number")
				}
				selectedJokerIDs = append(selectedJokerIDs, int(jokerIDFloat))
			}
		}

		// Now selectedJokerIDs is populated correctly regardless of source
		totalSelected += len(selectedJokerIDs)
	}

	// Parse selected vouchers if present
	if vouchersInterface, hasVouchers := selectionsMap["selectedVouchers"]; hasVouchers {

		if isCallFromBackend {
			// Backend calls pass in []int directly
			backendVouchers, ok := vouchersInterface.([]int)
			if !ok {
				return nil, fmt.Errorf("backend: selectedVouchers must be []int")
			}

			// Just use the voucher IDs directly
			selectedVoucherIDs = backendVouchers
		} else {
			// Frontend calls pass JSON that becomes []interface{}
			frontendVouchers, ok := vouchersInterface.([]interface{})
			if !ok {
				return nil, fmt.Errorf("frontend: selectedVouchers must be an array")
			}

			// Parse each voucher ID from the frontend format
			for _, voucherIDInterface := range frontendVouchers {
				voucherIDFloat, ok := voucherIDInterface.(float64)
				if !ok {
					return nil, fmt.Errorf("each selected voucher ID must be a number")
				}
				selectedVoucherIDs = append(selectedVoucherIDs, int(voucherIDFloat))
			}
		}

		// Now selectedVoucherIDs is populated correctly regardless of source
		totalSelected += len(selectedVoucherIDs)
	}

	// Check if they've selected too many items
	if totalSelected > item.MaxSelectable {
		return nil, fmt.Errorf("you can only select up to %d items from this pack", item.MaxSelectable)
	}

	// If nothing selected, reject the selection
	if totalSelected == 0 {
		return nil, fmt.Errorf("you must select at least one item from the pack")
	}

	// Now validate and process each type of selection based on the pack type
	switch item.PackType {
	case game_constants.PACK_TYPE_CARDS:
		// For card packs, verify selected cards
		if len(selectedCards) == 0 {
			return nil, fmt.Errorf("you must select at least one card from a cards pack")
		}
		if len(selectedJokerIDs) > 0 || len(selectedVoucherIDs) > 0 {
			return nil, fmt.Errorf("you can only select cards from a cards pack")
		}

		// Verify selected cards exist in the pack
		for _, selectedCard := range selectedCards {
			cardFound := false
			for _, card := range packContents.Cards {
				if card.Rank == selectedCard.Rank && card.Suit == selectedCard.Suit && card.Enhancement == selectedCard.Enhancement {
					cardFound = true
					break
				}
			}
			if !cardFound {
				return nil, fmt.Errorf("card %s of %s is not in the pack", selectedCard.Rank, selectedCard.Suit)
			}
		}

		// Add selected cards to player's inventory
		var purchasedCards []poker.Card
		if player.PurchasedPackCards != nil && len(player.PurchasedPackCards) > 0 {
			if err := json.Unmarshal(player.PurchasedPackCards, &purchasedCards); err != nil {
				return nil, fmt.Errorf("error parsing player's purchased cards: %v", err)
			}
		} else {
			purchasedCards = []poker.Card{}
		}

		purchasedCards = append(purchasedCards, selectedCards...)
		updatedPurchasedCardsJSON, err := json.Marshal(purchasedCards)
		if err != nil {
			return nil, fmt.Errorf("error updating purchased cards: %v", err)
		}
		player.PurchasedPackCards = updatedPurchasedCardsJSON

		log.Printf("[PROCESS PACK SELECTION] UPDATED selectedCards for player %s: %v", player.Username, selectedCards)

	case game_constants.PACK_TYPE_JOKERS:
		// For joker packs, verify selected jokers
		if len(selectedJokerIDs) == 0 {
			return nil, fmt.Errorf("you must select at least one joker from a jokers pack")
		}
		if len(selectedCards) > 0 || len(selectedVoucherIDs) > 0 {
			return nil, fmt.Errorf("you can only select jokers from a jokers pack")
		}

		// Verify selected jokers exist in the pack
		for _, selectedJokerID := range selectedJokerIDs {
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
				return nil, fmt.Errorf("joker ID %d is not in the pack", selectedJokerID)
			}
		}

		// Add selected jokers to player's inventory
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

		currentJokers.Juglares = append(currentJokers.Juglares, selectedJokerIDs...)
		updatedJokersJSON, err := json.Marshal(currentJokers)
		if err != nil {
			return nil, fmt.Errorf("error updating jokers: %v", err)
		}
		player.CurrentJokers = updatedJokersJSON

		log.Printf("[PROCESS PACK SELECTION] UPDATED selectedJokers for player %s: %v", player.Username, selectedJokerIDs)

	case game_constants.PACK_TYPE_VOUCHERS:
		// For voucher packs, verify selected vouchers
		if len(selectedVoucherIDs) == 0 {
			return nil, fmt.Errorf("you must select at least one voucher from a vouchers pack")
		}
		if len(selectedCards) > 0 || len(selectedJokerIDs) > 0 {
			return nil, fmt.Errorf("you can only select vouchers from a vouchers pack")
		}

		// Verify selected vouchers exist in the pack
		for _, selectedVoucherID := range selectedVoucherIDs {
			voucherFound := false
			for _, voucher := range packContents.Vouchers {
				if voucher.Value == selectedVoucherID {
					voucherFound = true
					break
				}
			}
			if !voucherFound {
				return nil, fmt.Errorf("voucher ID %d is not in the pack", selectedVoucherID)
			}
		}

		// Add selected vouchers to player's inventory
		var currentModifiers poker.Modifiers
		if player.Modifiers != nil && len(player.Modifiers) > 0 {
			if err := json.Unmarshal(player.Modifiers, &currentModifiers); err != nil {
				return nil, fmt.Errorf("error parsing player's modifiers: %v", err)
			}
		} else {
			currentModifiers = poker.Modifiers{
				Modificadores: []poker.Modifier{},
			}
		}

		// Create new modifier objects for each selected voucher
		for _, voucherID := range selectedVoucherIDs {
			newModifier := poker.Modifier{
				Value:    voucherID,
				LeftUses: -1, // Set to -1 if it doesn't expire until manually used
			}
			currentModifiers.Modificadores = append(currentModifiers.Modificadores, newModifier)
		}

		updatedModifiersJSON, err := json.Marshal(currentModifiers)
		if err != nil {
			return nil, fmt.Errorf("error updating modifiers: %v", err)
		}
		player.Modifiers = updatedModifiersJSON

		log.Printf("[PROCESS PACK SELECTION] UPDATED selectedVouchers for player %s: %v", player.Username, selectedVoucherIDs)
	}

	// Reset LastPurchasedPackItemId to prevent reuse
	player.LastPurchasedPackItemId = -1

	return player, nil
}

// Generate modifiers for voucher packs
func generatePackVouchers(rng *rand.Rand, count int) []poker.Modifier {
	vouchers := make([]poker.Modifier, count)

	for i := 0; i < count; i++ {
		// Simply generate a random number between 1 and 9
		modifierID := rng.Intn(9) + 1
		if modifierID == 0 {
			modifierID = 1
		}
		if modifierID == 2 || modifierID == 9 {
			modifierID = 3
		}

		// Create a modifier directly
		vouchers[i] = poker.Modifier{
			Value:    modifierID,
			LeftUses: -1, // Set to -1 if it doesn't expire until manually used
		}
	}

	return vouchers
}

func GetRerollPriceForPlayer(player *redis.InGamePlayer) int {
	// Calculate the reroll price based on the number of rerolls
	if player != nil {
		return player.Rerolls + 2
	}
	return -1
}

func GetGlobalShopRerollPrice(lobby *redis.GameLobby) int {
	// Calculate the global reroll price based on the number of rerolls
	if lobby != nil && lobby.ShopState != nil {
		return lobby.ShopState.Rerolls + 2
	}
	return -1
}

// RemovePurchasedItems removes shop items that have been purchased by the player
// It modifies the shopState directly and returns the modified pointer
func RemovePurchasedItems(shopState *redis.LobbyShop, player *redis.InGamePlayer) *redis.LobbyShop {
	// If player has no purchased items, return shopState unchanged
	if player == nil || player.CurrentShopPurchasedItemIDs == nil || len(player.CurrentShopPurchasedItemIDs) == 0 {
		return shopState
	}

	// Filter fixed packs
	filteredPacks := make([]redis.ShopItem, 0, len(shopState.FixedPacks))
	for _, item := range shopState.FixedPacks {
		if !player.CurrentShopPurchasedItemIDs[item.ID] {
			filteredPacks = append(filteredPacks, item)
		}
	}
	shopState.FixedPacks = filteredPacks

	// Filter fixed modifiers
	filteredModifiers := make([]redis.ShopItem, 0, len(shopState.FixedModifiers))
	for _, item := range shopState.FixedModifiers {
		if !player.CurrentShopPurchasedItemIDs[item.ID] {
			filteredModifiers = append(filteredModifiers, item)
		}
	}
	shopState.FixedModifiers = filteredModifiers

	// Filter jokers from ALL rerolls instead of just the latest one
	for rerollIndex := 0; rerollIndex < len(shopState.Rerolled); rerollIndex++ {
		// Since Jokers is a fixed-size array, we need to handle it differently
		// Create a copy of the jokers array
		var filteredJokers [3]redis.ShopItem
		for i, joker := range shopState.Rerolled[rerollIndex].Jokers {
			if player.CurrentShopPurchasedItemIDs[joker.ID] {
				// Mark as invalid by setting ID to -1 (frontend should not display items with ID < 0)
				filteredJokers[i] = redis.ShopItem{ID: -1}
			} else {
				// Keep the original joker
				filteredJokers[i] = joker
			}
		}

		// Update the jokers array in the shop state
		shopState.Rerolled[rerollIndex].Jokers = filteredJokers
	}

	return shopState
}
