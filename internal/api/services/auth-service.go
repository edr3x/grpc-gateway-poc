package services

import (
	"context"

	"go.uber.org/zap"
)

type AuthService struct {
	logger *zap.Logger
}

func NewAuthService() *AuthService {
	return &AuthService{
		logger: zap.L(),
	}
}

func (s *AuthService) Login(ctx context.Context) error {
	// TODO: call this service for business logic
	return nil
}
