package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/zishang520/socket.io/v2/socket"
)

type ClientInfo struct {
	Socket   *socket.Socket
	ClientID string
	Name     string
}

type Server struct {
	clients map[string]*ClientInfo
	mutex   sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]*ClientInfo),
	}
}

func (s *Server) AddClient(clientID, name string, socket *socket.Socket) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clients[clientID] = &ClientInfo{
		Socket:   socket,
		ClientID: clientID,
		Name:     name,
	}
	log.Printf("Client registered: ID=%s, Name=%s, SocketID=%s", clientID, name, socket.Id())
}

func (s *Server) RemoveClient(clientID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if client, exists := s.clients[clientID]; exists {
		log.Printf("Client unregistered: ID=%s, Name=%s", clientID, client.Name)
		delete(s.clients, clientID)
	}
}

func (s *Server) GetClient(clientID string) (*ClientInfo, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	client, exists := s.clients[clientID]
	return client, exists
}

func (s *Server) GetAllClients() map[string]*ClientInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[string]*ClientInfo)
	for k, v := range s.clients {
		result[k] = v
	}
	return result
}

func (s *Server) SendToClient(clientID, message string) bool {
	if client, exists := s.GetClient(clientID); exists {
		client.Socket.Emit("selective_message", map[string]any{
			"message": message,
			"from":    "server",
			"to":      clientID,
		})
		return true
	}
	return false
}

func (s *Server) SendToRandomClients(message string, count int) {
	clients := s.GetAllClients()
	if len(clients) == 0 {
		return
	}

	clientIDs := make([]string, 0, len(clients))
	for id := range clients {
		clientIDs = append(clientIDs, id)
	}

	if count > len(clientIDs) {
		count = len(clientIDs)
	}

	rand.Shuffle(len(clientIDs), func(i, j int) {
		clientIDs[i], clientIDs[j] = clientIDs[j], clientIDs[i]
	})

	for i := 0; i < count; i++ {
		clientID := clientIDs[i]
		s.SendToClient(clientID, message)
		log.Printf("Sent selective message to client: %s", clientID)
	}
}

func (s *Server) StartPeriodicMessaging() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		messageCounter := 1
		for range ticker.C {
			clients := s.GetAllClients()
			if len(clients) > 0 {
				message := "Periodic message #" + strconv.Itoa(messageCounter) + " from server"
				
				numClientsToMessage := rand.Intn(len(clients)) + 1
				s.SendToRandomClients(message, numClientsToMessage)
				log.Printf("Sent periodic message #%d to %d random clients", messageCounter, numClientsToMessage)
				messageCounter++
			}
		}
	}()
}

func main() {
	server := socket.NewServer(nil, nil)
	appServer := NewServer()

	appServer.StartPeriodicMessaging()

	server.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		log.Printf("Socket connected: %s", client.Id())

		client.On("register", func(datas ...any) {
			data := datas[0].(map[string]any)
			clientID := data["client_id"].(string)
			name := data["name"].(string)
			
			appServer.AddClient(clientID, name, client)
			
			client.Emit("registered", map[string]any{
				"status":    "success",
				"client_id": clientID,
				"message":   "Successfully registered with server",
			})
		})

		client.On("message", func(datas ...any) {
			message := datas[0].(string)
			log.Printf("Received message from socket %s: %s", client.Id(), message)
			
			client.Emit("response", "Server received: "+message)
		})

		client.On("ping", func(datas ...any) {
			log.Printf("Received ping from socket: %s", client.Id())
			client.Emit("pong", "Server pong response")
		})

		client.On("disconnect", func(datas ...any) {
			clients := appServer.GetAllClients()
			for clientID, clientInfo := range clients {
				if clientInfo.Socket.Id() == client.Id() {
					appServer.RemoveClient(clientID)
					break
				}
			}
			log.Printf("Socket disconnected: %s", client.Id())
		})

		client.Emit("welcome", "Welcome to Socket.IO server! Please register with your client ID.")
	})

	http.Handle("/socket.io/", server.ServeHandler(nil))

	log.Println("Socket.IO server starting on :8080")
	log.Println("Server will send periodic messages to random clients every 10 seconds")
	log.Fatal(http.ListenAndServe(":8080", nil))
}