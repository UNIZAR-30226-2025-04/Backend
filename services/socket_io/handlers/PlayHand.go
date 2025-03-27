package handlers

import (
	"Nogler/services/poker"
	"Nogler/services/redis"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

// Hacer aquí una tablica de relación nombre -> puntos o devolver desde el otro lado
// Un valor directamente. Lo mejor sería que consultemos en un spot (redis pg o dnd  sea)
// El nivel al que tenemo sla mano para saber fichas y mult base
// Ahora mismo está como string en el aproach mencionado sería 2 ints, fichas y mult
func PlayHand(redisClient *redis.RedisClient, client *socket.Socket,
	db *gorm.DB, username string) func(args ...interface{}) {
	return func(args ...interface{}) {

		log.Printf("PlayHand iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[HAND-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta la mano a jugar"})
			return
		}

		// TODO: y si el mismo usuario vuelve a intentar conectarse otra vez sin haberse desconectado?

		handData := args[0].(map[string]interface{}) // Argument is expected to be a map (which is a generic object)
		handJson, err := json.Marshal(handData)      // Convert the argument to JSON
		if err != nil {
			log.Printf("[HAND-ERROR] Error al convertir la mano a JSON: %v", err)
			client.Emit("error", gin.H{"error": "Error al convertir la mano"})
			return
		}

		// Parse the JSON into the poker.Hand struct
		var hand poker.Hand
		err = json.Unmarshal(handJson, &hand)
		if err != nil {
			log.Printf("[HAND-ERROR] Error al parsear la mano: %v", err)
			client.Emit("error", gin.H{"error": "Error al procesar la mano"})
			return
		}

		// Calculate base points
		fichas, mult := poker.BestHand(hand)

		// Apply jokers (passing the hand which contains the jokers)
		finalFichas, finalMult := poker.ApplyJokers(hand, hand.Jokers, fichas, mult)
		valorFinal := finalFichas * finalMult

		// Log the result
		log.Println("Jugador ha puntuado la friolera de:", valorFinal)
		// Emit success response
		client.Emit("played_hand", gin.H{
			"points":  valorFinal,
			"message": "¡Mano jugada con éxito!",
		})

		//logear en redis + pg cuanto ha puntuado supongo IMPORTANTEEEEEEEEEEEEEEEEEEEEEE

		// Añadir aquí tajo checks, está en lobby redis + postgres + tod ala pesca. si se pueden hacer en asincrono mejor, así no esperamos a ello.
		// 0. Check if user is in lobby (Postgres)
		/*
			isInLobby, err := utils.IsPlayerInLobby(db, lobbyID, username)
			if err != nil {
				fmt.Println("Database error:", err)
				client.Emit("error", gin.H{"error": "Database error"})
				return
			}

			if !isInLobby {
				fmt.Println("User is NOT in lobby:", username, "Lobby:", lobbyID)
				client.Emit("error", gin.H{"error": "You must join the lobby before sending messages"})
				return
			}
		*/

	}
}

func ApplyJokers(h poker.Hand, fichas int, mult int) int {
	// Given a hand and the points obtained from poker.Hand
	return fichas * mult
}
