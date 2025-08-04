package services

import (
	"github.com/google/uuid"

	"ccagent/models"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

func (s *SessionService) GenerateSession() *models.Session {
	return &models.Session{
		ID: uuid.New().String(),
	}
}
