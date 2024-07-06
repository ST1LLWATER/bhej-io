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
	Name string `json:"name"`
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
	clients  = make(map[string]map[*websocket.Conn]User)
	cliMutex sync.Mutex
)

func IndexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func RoomHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "room.html", nil)
}

func WebsocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer conn.Close()

	roomID := c.Param("id")
	name := c.Query("name")
	userID := uuid.New().String()
	user := User{ID: userID, Addr: conn.RemoteAddr().String(), Name: name}

	log.Println("inserting room to clients:", roomID)

	cliMutex.Lock()
	if _, ok := clients[roomID]; !ok {
		clients[roomID] = make(map[*websocket.Conn]User)
	}
	clients[roomID][conn] = user
	cliMutex.Unlock()

	broadcastUsers(roomID)

	defer func() {
		cliMutex.Lock()
		delete(clients[roomID], conn)
		if len(clients[roomID]) == 0 {
			delete(clients, roomID)
		}
		cliMutex.Unlock()
		broadcastUsers(roomID)
	}()

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message: ", err)
			break
		}

		if messageType == websocket.BinaryMessage {
			handleFileMessage(roomID, conn, msg)
		}
	}
}

func broadcastUsers(roomID string) {
	cliMutex.Lock()
	defer cliMutex.Unlock()

	users := make([]User, 0, len(clients[roomID]))
	for _, user := range clients[roomID] {
		users = append(users, user)
	}

	userList, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling users: ", err)
		return
	}

	for conn := range clients[roomID] {
		err := conn.WriteMessage(websocket.TextMessage, userList)
		if err != nil {
			conn.Close()
			delete(clients[roomID], conn)
		}
	}
}

func UsersHandler(c *gin.Context) {
	cliMutex.Lock()
	defer cliMutex.Unlock()

	roomID := c.Param("id")

	if roomClients, ok := clients[roomID]; ok {
		users := make([]User, 0, len(roomClients))
		for _, user := range roomClients {
			users = append(users, user)
		}
		c.JSON(http.StatusOK, users)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
	}
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

func handleFileMessage(roomID string, sender *websocket.Conn, fileData []byte) {
	log.Println("Handling file message")
	var fileMessage FileMessage
	err := json.Unmarshal(fileData, &fileMessage)
	if err != nil {
		log.Println("Error unmarshalling file message: ", err)
		return
	}

	fileMessage.SenderID = clients[roomID][sender].ID

	dataToSend, err := json.Marshal(fileMessage)
	if err != nil {
		log.Println("Error marshalling file message: ", err)
		return
	}

	cliMutex.Lock()
	defer cliMutex.Unlock()

	var wg sync.WaitGroup

	for conn := range clients[roomID] {
		wg.Add(1)
		go func(conn *websocket.Conn) {
			defer wg.Done()
			err := conn.WriteMessage(websocket.BinaryMessage, dataToSend)
			if err != nil {
				log.Println("Error sending message to client: ", err)
				conn.Close()
				delete(clients[roomID], conn)
			}
		}(conn)
	}

	wg.Wait()
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
