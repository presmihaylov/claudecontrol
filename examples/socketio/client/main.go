package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/zishang520/engine.io-client-go/transports"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/engine.io/v2/utils"
	"github.com/zishang520/socket.io-client-go/socket"
)

type Client struct {
	ID       string
	Name     string
	Socket   *socket.Socket
	IsActive bool
}

func NewClient(id, name string) *Client {
	return &Client{
		ID:       id,
		Name:     name,
		IsActive: false,
	}
}

func (c *Client) Connect() {
	opts := socket.DefaultOptions()
	opts.SetTransports(types.NewSet(transports.Polling, transports.WebSocket))

	manager := socket.NewManager("http://localhost:8080", opts)

	manager.On("error", func(errs ...any) {
		utils.Log().Warning("[%s] Manager Error: %v", c.ID, errs)
	})

	manager.On("ping", func(...any) {
		utils.Log().Warning("[%s] Manager Ping", c.ID)
	})

	manager.On("reconnect", func(...any) {
		utils.Log().Warning("[%s] Manager Reconnected", c.ID)
		c.register()
	})

	manager.On("reconnect_attempt", func(...any) {
		utils.Log().Warning("[%s] Manager Reconnect Attempt", c.ID)
	})

	manager.On("reconnect_error", func(errs ...any) {
		utils.Log().Warning("[%s] Manager Reconnect Error: %v", c.ID, errs)
	})

	manager.On("reconnect_failed", func(errs ...any) {
		utils.Log().Warning("[%s] Manager Reconnect Failed: %v", c.ID, errs)
	})

	c.Socket = manager.Socket("/", opts)

	c.Socket.On("connect", func(args ...any) {
		utils.Log().Success("[%s] Connected to server, socket ID: %v", c.ID, c.Socket.Id())
		c.IsActive = true
		
		utils.SetTimeout(func() {
			c.register()
		}, 500*time.Millisecond)
		
		utils.Log().Warning("[%s] connect %v", c.ID, args)
	})

	c.Socket.On("connect_error", func(args ...any) {
		utils.Log().Warning("[%s] connect_error %v", c.ID, args)
		c.IsActive = false
	})

	c.Socket.On("disconnect", func(args ...any) {
		utils.Log().Warning("[%s] disconnect: %+v", c.ID, args)
		c.IsActive = false
	})

	c.Socket.On("welcome", func(args ...any) {
		utils.Log().Question("[%s] Server welcome: %+v", c.ID, args)
	})

	c.Socket.On("registered", func(args ...any) {
		data := args[0].(map[string]any)
		utils.Log().Success("[%s] Registration successful: %+v", c.ID, data)
		
		c.startPeriodicPing()
		c.startRandomMessages()
	})

	c.Socket.On("response", func(args ...any) {
		utils.Log().Question("[%s] Server response: %+v", c.ID, args)
	})

	c.Socket.On("pong", func(args ...any) {
		utils.Log().Question("[%s] Server pong: %+v", c.ID, args)
	})

	c.Socket.On("selective_message", func(args ...any) {
		data := args[0].(map[string]any)
		utils.Log().Success("[%s] 🎯 SELECTIVE MESSAGE: %s (from: %s, to: %s)", 
			c.ID, data["message"], data["from"], data["to"])
	})

	c.Socket.OnAny(func(args ...any) {
		utils.Log().Info("[%s] OnAny: %+v", c.ID, args)
	})
}

func (c *Client) register() {
	if !c.IsActive {
		return
	}
	
	registrationData := map[string]any{
		"client_id": c.ID,
		"name":      c.Name,
	}
	
	c.Socket.Emit("register", registrationData)
	utils.Log().Info("[%s] Sent registration data: %+v", c.ID, registrationData)
}

func (c *Client) startPeriodicPing() {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			if !c.IsActive {
				continue
			}
			c.Socket.Emit("ping", types.NewStringBufferString(fmt.Sprintf("ping from client %s", c.ID)))
			utils.Log().Info("[%s] Sent ping to server", c.ID)
		}
	}()
}

func (c *Client) startRandomMessages() {
	go func() {
		ticker := time.NewTicker(time.Duration(rand.Intn(20)+10) * time.Second)
		defer ticker.Stop()
		
		messageCounter := 1
		for range ticker.C {
			if !c.IsActive {
				continue
			}
			message := fmt.Sprintf("Random message #%d from client %s", messageCounter, c.ID)
			c.Socket.Emit("message", types.NewStringBufferString(message))
			utils.Log().Info("[%s] Sent random message: %s", c.ID, message)
			messageCounter++
		}
	}()
}

func main() {
	var clientID, clientName string
	
	if len(os.Args) >= 3 {
		clientID = os.Args[1]
		clientName = os.Args[2]
	} else {
		rand.Seed(time.Now().UnixNano())
		clientID = "client-" + strconv.Itoa(rand.Intn(10000))
		clientName = "TestClient-" + strconv.Itoa(rand.Intn(1000))
	}

	utils.Log().Success("Starting client with ID: %s, Name: %s", clientID, clientName)
	
	client := NewClient(clientID, clientName)
	client.Connect()

	select {}
}