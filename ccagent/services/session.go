package services

import (
	"ccagent/core"
	"ccagent/models"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

func (s *SessionService) GenerateSession() *models.Session {
	sessionID := core.NewID("sess")
	return &models.Session{
		ID: sessionID,
	}
}
