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
	sessionID, err := core.NewID("sess")
	if err != nil {
		// Since this function doesn't return an error, we need to handle this case
		// Generate a fallback ID or panic. For now, we'll use a basic fallback
		sessionID = "sess_unknown"
	}
	return &models.Session{
		ID: sessionID,
	}
}
