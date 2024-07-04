package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type User struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

type FileMessage struct {
	FileName    string `json:"file_name"`
	ChunkIndex  int    `json:"chunk_index"`
	FileSize    int    `json:"file_size"`
	ChunkData   []byte `json:"chunk_data"`
	TotalChunks int    `json:"total_chunks"`        // Add this field to hold the total number of chunks
	SenderID    string `json:"sender_id,omitempty"` // Optional: include only in response
}

var (
	clients  = make(map[*websocket.Conn]User)
	cliMutex sync.Mutex
)

func IndexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func WebsocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer conn.Close()

	userID := uuid.New().String()
	user := User{ID: userID, Addr: conn.RemoteAddr().String()}

	cliMutex.Lock()
	clients[conn] = user
	cliMutex.Unlock()

	broadcastUsers()

	defer func() {
		cliMutex.Lock()
		delete(clients, conn)
		cliMutex.Unlock()
		broadcastUsers()
	}()

	for {
		messageType, msg, err := conn.ReadMessage()
		log.Println("MESSAGE TYPE: ", messageType)
		if err != nil {
			log.Println("Error reading message: ", err)
			break
		}

		// if messageType == websocket.BinaryMessage {
		log.Println("RECEIVING FILE BINARY MESSAGE")
		handleFileMessage(conn, msg)
		// }

	}
}

func broadcastUsers() {
	cliMutex.Lock()

	defer cliMutex.Unlock()

	users := make([]User, 0, len(clients))

	for _, user := range clients {
		users = append(users, user)
	}

	userList, err := json.Marshal(users)

	if err != nil {
		log.Println("Error marshalling users: ", err)
		return
	}

	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, userList)
		if err != nil {
			conn.Close()
			delete(clients, conn)
		}
	}
}

func UsersHandler(c *gin.Context) {
	cliMutex.Lock()
	defer cliMutex.Unlock()

	users := make([]User, 0, len(clients))

	for _, user := range clients {
		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

// func handleFileMessage(sender *websocket.Conn, fileData []byte) {
// 	log.Println("Handling file message")
// 	var fileMessage FileMessage
// 	err := json.Unmarshal(fileData, &fileMessage)
// 	if err != nil {
// 		log.Println("Error unmarshalling file message: ", err)
// 		return
// 	}

// 	// Set the sender ID from the connection's associated client data
// 	if clientData, ok := clients[sender]; ok {
// 		fileMessage.SenderID = clientData.ID
// 	}

// 	// Prepare to send the complete message, including total_chunks, to other clients
// 	dataToSend, err := json.Marshal(fileMessage)
// 	if err != nil {
// 		log.Println("Error marshalling file message: ", err)
// 		return
// 	}

// 	cliMutex.Lock()
// 	defer cliMutex.Unlock()

// 	// Send the updated file message to all clients (except the sender, optionally)
// 	for conn := range clients {
// 		// if conn != sender {
// 		err := conn.WriteMessage(websocket.BinaryMessage, dataToSend)
// 		if err != nil {
// 			log.Println("Error sending message to client: ", err)
// 			conn.Close()
// 			delete(clients, conn)
// 		}
// 		// }
// 	}
// }

func handleFileMessage(sender *websocket.Conn, fileData []byte) {
	log.Println("Handling file message")
	var fileMessage FileMessage
	err := json.Unmarshal(fileData, &fileMessage)
	if err != nil {
		log.Println("Error unmarshalling file message: ", err)
		return
	}

	fileMessage.SenderID = clients[sender].ID

	dataToSend, err := json.Marshal(fileMessage)
	if err != nil {
		log.Println("Error marshalling file message: ", err)
		return
	}

	cliMutex.Lock()
	defer cliMutex.Unlock()

	// Create a wait group to ensure all goroutines finish
	var wg sync.WaitGroup

	for conn := range clients {
		wg.Add(1) // Increment the WaitGroup counter
		go func(conn *websocket.Conn) {
			defer wg.Done() // Decrement the counter when the goroutine completes
			err := conn.WriteMessage(websocket.BinaryMessage, dataToSend)
			if err != nil {
				log.Println("Error sending message to client: ", err)
				conn.Close()
				delete(clients, conn)
			}
		}(conn)
	}

	wg.Wait() // Wait here until all goroutines finish
}

// func handleFileMessage(sender *websocket.Conn, fileData []byte) {
// 	log.Println("Handling file message")
// 	var fileMessage FileMessage
// 	err := json.Unmarshal(fileData, &fileMessage)
// 	if err != nil {
// 		log.Println("Error unmarshalling file message: ", err)
// 		return
// 	}

// 	fileMessage.SenderID = clients[sender].ID

// 	dataToSend, err := json.Marshal(fileMessage)
// 	if err != nil {
// 		log.Println("Error marshalling file message: ", err)
// 		return
// 	}

// 	cliMutex.Lock()
// 	defer cliMutex.Unlock()

// 	for conn := range clients {
// 		err := conn.WriteMessage(websocket.BinaryMessage, dataToSend)
// 		if err != nil {
// 			log.Println("Error sending message to client: ", err)
// 			conn.Close()
// 			delete(clients, conn)
// 		}
// 	}
// }
