package main

type UnknownMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type PingPayload struct{}

type PongPayload struct{}
