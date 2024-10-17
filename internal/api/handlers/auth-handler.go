package handlers

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/edr3x/gateway-impl/pkg/proto"
)

type AuthHandler struct {
	logger *zap.Logger
	pb.AuthServiceServer
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		logger: zap.L(),
	}
}

func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (res *pb.LoginResponse, err error) {
	email := req.Email
	password := req.Password

	if email != "test@email.com" {
		return nil, status.Error(codes.NotFound, "email not found")
	}

	if password != "Password@123" {
		return nil, status.Error(codes.Unauthenticated, "wrong credentials provided")
	}

	res = &pb.LoginResponse{
		AccessToken:  "asdfasdrqawerfasdgasdf",
		RefreshToken: "asdawe3rasdgasdradvbasedrasdadssfghas",
	}

	return res, nil
}
