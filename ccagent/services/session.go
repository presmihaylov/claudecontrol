package services

import (
	"ccagent/models"
	"github.com/google/uuid"
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
