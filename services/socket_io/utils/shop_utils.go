package socketio_utils

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
)

func GenerateSeed(parts ...interface{}) int64 {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprint(parts...)))
	return int64(h.Sum64())
}

func CalculatePackPrice(numItems int) int {
	return numItems + 1
}

// Change this OBVIOUSLY GPT GENERATED for a real one
func RandomModifierType(rng *rand.Rand) string {
	return "modifier yeahhhhh"
}
