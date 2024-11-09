package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Global variables for the socket server and MongoDB client
var socketServer *socketio.Server
var mongoClient *mongo.Client
var messagesCollection *mongo.Collection

var allowOriginFunc = func(r *http.Request) bool {
	return true
}

// Message struct represents a chat message
type Message struct {
	ID      string    `bson:"_id,omitempty"`
	Sender  string    `bson:"sender"`
	Content string    `bson:"content"`
	Time    time.Time `bson:"time"`
}

// MongoDB connection function
func connectToMongoDB() {
	var err error
	// Replace with your MongoDB URI
	clientOptions := options.Client().ApplyURI("mongodb+srv://admin:admin@admin.qboa2og.mongodb.net/socket")
	mongoClient, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	// Verify connection
	err = mongoClient.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("MongoDB ping error:", err)
	}

	// Set collection for messages
	messagesCollection = mongoClient.Database("socket").Collection("messages")
	log.Println("Connected to MongoDB!")
}

// Exported function that will be called by Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Serve the Socket.IO requests
	socketServer.ServeHTTP(w, r)
}

func main() {
	// Connect to MongoDB
	connectToMongoDB()

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

		// Save the message to MongoDB
		message := Message{
			Sender:  s.ID(), // Store sender ID as an example
			Content: msg,
			Time:    time.Now(),
		}

		_, err := messagesCollection.InsertOne(context.TODO(), message)
		if err != nil {
			log.Println("Failed to save message:", err)
		} else {
			log.Println("Message saved to MongoDB")
		}
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
