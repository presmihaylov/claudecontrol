package clients

// MessageSender defines the interface for sending messages to WebSocket clients
type MessageSender interface {
	SendMessage(clientID string, msg any) error
}
