package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

// Global variable for the socket server
var socketServer *socketio.Server

var allowOriginFunc = func(r *http.Request) bool {
	return true
}

// Exported function that will be called by Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Serve the Socket.IO requests
	socketServer.ServeHTTP(w, r)
}

func main() {
	// Initialize the Socket.IO server
	socketServer = socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	})

	// Handle connection
	socketServer.OnConnect("/", func(s socketio.Conn) error {
		log.Println("Connected:", s.ID())
		s.SetContext("")
		s.Join("chat")
		return nil
	})

	// Handle chat message
	socketServer.OnEvent("/", "message", func(s socketio.Conn, msg string) {
		log.Println("Message received:", msg)
		socketServer.BroadcastToRoom("/", "chat", "message", msg)
	})

	// Handle disconnection
	socketServer.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("Disconnected:", s.ID(), "Reason:", reason)
	})

	go socketServer.Serve()
	defer socketServer.Close()

	// Create a new Gin router
	router := gin.Default()

	// Serve Socket.IO at a specific path
	router.GET("/socket.io/*any", gin.WrapH(socketServer))
	router.POST("/socket.io/*any", gin.WrapH(socketServer))

	// Start the server
	// Note: This line won't be executed when deployed on Vercel
	router.Run(":4000")
}
