package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"nhooyr.io/websocket"
)

// Message struct to represent chat messages
type Message struct {
	Sender   string    `json:"sender" bson:"sender"`
	Receiver string    `json:"receiver" bson:"receiver"`
	Content  string    `json:"content" bson:"content"`
	Time     time.Time `json:"time" bson:"time"`
}

var (
	connections   map[string]*websocket.Conn // Map to store WebSocket connections for each user
	messageQueues map[string][]Message       // Map to store message queues for each user
	mongoClient   *mongo.Client              // MongoDB client instance
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "myapp_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "status"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
}

func connectMongo(ctx context.Context) {
	mongoURI := "mongodb://mymongo:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Error creating MongoDB client:", err)
	}
	mongoClient = client
}

func main() {
	// Establish MongoDB connection
	ctx := context.Background()
	connectMongo(ctx)
	// Initialize connections map and message queues map
	connections = make(map[string]*websocket.Conn)
	messageQueues = make(map[string][]Message)
	// Handle WebSocket connections
	http.HandleFunc("/ws", handleWebSocket)
	// Expose Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	// Start HTTP server
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	requestsTotal.WithLabelValues(r.Method, "200").Inc()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		ctx := r.Context()

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Println("Error accepting WebSocket:", err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "Internal Server Error")

		// Read user ID from request (e.g., from URL query parameter or header)
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			log.Println("User ID not provided in request")
			return
		}
		log.Println("User with ID ", userID, " connected")
		// Add the new connection to the connections map
		connections[userID] = conn
		// Retrieve and send offline messages, if any
		for _, msg := range messageQueues[userID] {
			sendMessage(msg)
		}
		// Clear the message queue for the user
		messageQueues[userID] = nil

		// Handle disconnections and reconnections
		for {
			// Read message from client
			_, msg, err := conn.Read(ctx)
			if err != nil {
				log.Println("Error reading message from client:", err)
				break
			}

			// Unmarshal JSON message into Message struct
			var message Message
			err = json.Unmarshal(msg, &message)
			if err != nil {
				log.Println("Error unmarshaling JSON:", err)
				continue
			}

			// Store message in MongoDB
			err = saveMessage(message)
			if err != nil {
				log.Println("Error saving message to MongoDB:", err)
			}

			// Send message to recipient user
			sendMessage(message)
		}

		// Remove the disconnected connection from the connections map
		delete(connections, userID)
	}()
	wg.Wait()
}

func sendMessage(message Message) {
	// Get recipient's WebSocket connection from the connections map
	conn, ok := connections[message.Receiver]
	if !ok {
		// Recipient is offline, enqueue the message in their message queue
		messageQueues[message.Receiver] = append(messageQueues[message.Receiver], message)
		log.Printf("Recipient %s is not connected, message enqueued\n", message.Receiver)
		return
	}

	// Marshal message to JSON
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		return
	}

	// Send message to recipient's WebSocket connection
	err = conn.Write(context.Background(), websocket.MessageText, msgBytes)
	if err != nil {
		log.Println("Error sending message to recipient:", err)
		return
	}

	log.Printf("Message sent to %s: %+v\n", message.Receiver, message)
}

func saveMessage(message Message) error {
	// Get MongoDB collection instance
	collection := mongoClient.Database("chat").Collection("messages")

	// Insert message into MongoDB collection
	_, err := collection.InsertOne(context.Background(), message)
	if err != nil {
		return err
	}

	log.Printf("Saved message: %+v\n", message)
	return nil
}
