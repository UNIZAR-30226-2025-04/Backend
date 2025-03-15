package socket_io

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io/v2/socket"

	"Nogler/services/redis"
)

var (
	isTestMode bool = false
	logFile    *os.File
	logger     *log.Logger
)

func initLogger() error {
	// Crear directorio logs si no existe
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio de logs: %v", err)
	}

	// Crear o abrir archivo de log con timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("socket_io_%s.log", timestamp))
	
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error abriendo archivo de log: %v", err)
	}

	logFile = file
	// Configurar logger para escribir tanto en archivo como en stdout
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger = log.New(multiWriter, "", log.LstdFlags)
	return nil
}

type SocketServer struct {
	sio_server  *socket.Server
	redisClient *redis.RedisClient
}

func (sio *SocketServer) Start(router *gin.Engine, db *gorm.DB, redisClient *redis.RedisClient) {
	// Inicializar logger
	if err := initLogger(); err != nil {
		log.Fatalf("[SOCKET-ERROR] Error inicializando logger: %v", err)
	}
	// No cerramos el archivo de log aquí para evitar problemas

	if db == nil {
		logger.Println("[SOCKET-CONFIG] Modo test activado")
		isTestMode = true
	}
	sio.redisClient = redisClient

	logger.Println("[SOCKET-CONFIG] Configurando opciones del servidor...")
	c := socket.DefaultServerOptions()
	c.SetServeClient(true)
	c.SetPingInterval(25000 * time.Millisecond)
	c.SetPingTimeout(20000 * time.Millisecond)
	c.SetMaxHttpBufferSize(1000000)
	c.SetConnectTimeout(45000 * time.Millisecond)
	c.SetTransports(types.NewSet("polling", "websocket"))
	c.SetAllowUpgrades(true)
	c.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	sio.sio_server = socket.NewServer(nil, c)
	
	// Añadir log para eventos del servidor
	sio.sio_server.On("error", func(err ...interface{}) {
		logger.Printf("[SOCKET-ERROR] Error en el servidor: %v", err)
	})

	// Manejar conexiones
	sio.sio_server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)
		
		// Simplemente usar el ID del cliente para identificarlo
		logger.Printf("[SOCKET-INFO] Nueva conexión: ID: %s", client.Id())
		
		// Manejar evento join_lobby
		client.On("join_lobby", func(args ...interface{}) {
			logger.Printf("[SOCKET-DEBUG] join_lobby recibido: %+v", args)
			
			if len(args) < 1 {
				logger.Printf("[SOCKET-ERROR] Faltan argumentos para join_lobby")
				client.Emit("error", gin.H{"error": "Falta el ID del lobby"})
				return
			}

			lobbyID, ok := args[0].(string)
			if !ok {
				logger.Printf("[SOCKET-ERROR] Tipo de argumento inválido: %T", args[0])
				client.Emit("error", gin.H{"error": "Formato de lobby_id inválido"})
				return
			}

			// Emitir respuesta simple
			logger.Printf("[SOCKET-DEBUG] Emitiendo lobby_joined para lobby %s", lobbyID)
			client.Emit("lobby_joined", gin.H{
				"lobby_id": lobbyID,
				"message": "¡Bienvenido al lobby!",
			})
			logger.Printf("[SOCKET-DEBUG] Respuesta enviada correctamente")
		})

		// Manejar desconexión
		client.On("disconnect", func(reason ...interface{}) {
			logger.Printf("[SOCKET-DEBUG] Cliente desconectado: %v", reason)
		})
	})

	logger.Println("[SOCKET-CONFIG] Configurando rutas HTTP...")
	router.POST("/socket.io/*f", gin.WrapH(sio.sio_server.ServeHandler(nil)))
	router.GET("/socket.io/*f", gin.WrapH(sio.sio_server.ServeHandler(nil)))

	logger.Println("[SOCKET] Servidor Socket.IO iniciado exitosamente")
}
