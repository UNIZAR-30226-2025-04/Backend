package handlers

import (
	"Nogler/services/redis"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/socket.io/v2/socket"
)

func HandleJoinLobby(redisClient *redis.RedisClient, client *socket.Socket,
	username string, args ...interface{}) func(args ...interface{}) {
	return func(args ...interface{}) {
		log.Printf("[JOIN] HandleJoinLobby iniciado - Usuario: %s, Args: %v, Socket ID: %s",
			username, args, client.Id())

		if len(args) < 1 {
			log.Printf("[JOIN-ERROR] Faltan argumentos para usuario %s", username)
			client.Emit("error", gin.H{"error": "Falta el ID del lobby"})
			return
		}

		lobbyID := args[0].(string)
		log.Printf("[JOIN] Procesando lobby ID: %s para usuario: %s", lobbyID, username)

		// 1. Verificar si existe el lobby en Redis
		lobby, err := redisClient.GetGameLobby(lobbyID)
		if err != nil {
			log.Println("Error al buscar lobby:", err)
			client.Emit("error", gin.H{"error": "Lobby no encontrado en Redis"})
			return
		}
		log.Println("Lobby encontrado:", lobby)

		// 2. Verificar si el jugador ya está en el lobby
		currentLobby, err := redisClient.GetPlayerCurrentLobby(username)
		if err == nil && currentLobby == lobbyID {
			client.Emit("error", gin.H{"error": "Ya estás en este lobby"})
			return
		}
		log.Println("Jugador no está en el lobby:", username, currentLobby)
		// 3. Crear objeto InGamePlayer con valores iniciales
		player := &redis.InGamePlayer{
			Username:       username,
			LobbyId:        lobbyID,
			PlayersMoney:   1000,
			CurrentDeck:    json.RawMessage(`{"cards":[]}`),
			Modifiers:      json.RawMessage(`{}`),
			CurrentJokers:  json.RawMessage(`{}`),
			MostPlayedHand: json.RawMessage(`{}`),
		}
		log.Println("Jugador creado:", player)
		// 4. Guardar estado del jugador en Redis
		if err := redisClient.SaveInGamePlayer(player); err != nil {
			log.Println("Error al guardar jugador en Redis:", err)
			client.Emit("error", gin.H{"error": "Error al unirse al lobby"})
			return
		}
		log.Println("Jugador guardado en Redis")
		// 5. Unir al socket room
		client.Join(socket.Room(lobbyID))
		log.Println("Jugador unido al room:", lobbyID)
		// 6. Notificar éxito
		log.Printf("[JOIN-SUCCESS] Usuario %s unido exitosamente al lobby %s", username, lobbyID)
		client.Emit("lobby_joined", gin.H{
			"lobby_id":         lobbyID,
			"message":          "¡Bienvenido al lobby!",
			"total_points":     lobby.TotalPoints,
			"number_of_rounds": lobby.NumberOfRounds,
		})
	}
}
