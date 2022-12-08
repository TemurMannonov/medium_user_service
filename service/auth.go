package service

import (
	"context"
	"encoding/json"
	"time"

	pb "github.com/TemurMannonov/medium_user_service/genproto/user_service"
	"github.com/TemurMannonov/medium_user_service/pkg/utils"
	"github.com/TemurMannonov/medium_user_service/storage/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/TemurMannonov/medium_user_service/storage"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	storage  storage.StorageI
	inMemory storage.InMemoryStorageI
}

func NewAuthService(strg storage.StorageI, inMemory storage.InMemoryStorageI) *AuthService {
	return &AuthService{
		storage:  strg,
		inMemory: inMemory,
	}
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*emptypb.Empty, error) {
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error: %v", err)
	}

	user := repo.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Type:      repo.UserTypeUser,
		Password:  hashedPassword,
	}

	userData, err := json.Marshal(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error: %v", err)
	}

	err = s.inMemory.Set("user_"+user.Email, string(userData), 10*time.Minute)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error: %v", err)
	}

	return &emptypb.Empty{}, nil
}
