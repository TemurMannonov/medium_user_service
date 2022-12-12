package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TemurMannonov/medium_user_service/config"
	"github.com/TemurMannonov/medium_user_service/genproto/notification_service"
	pb "github.com/TemurMannonov/medium_user_service/genproto/user_service"
	"github.com/TemurMannonov/medium_user_service/pkg/utils"
	"github.com/TemurMannonov/medium_user_service/storage/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	grpcPkg "github.com/TemurMannonov/medium_user_service/pkg/grpc_client"
	"github.com/TemurMannonov/medium_user_service/storage"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	storage    storage.StorageI
	inMemory   storage.InMemoryStorageI
	grpcClient grpcPkg.GrpcClientI
	cfg        *config.Config
}

func NewAuthService(strg storage.StorageI, inMemory storage.InMemoryStorageI, grpcConn grpcPkg.GrpcClientI, cfg *config.Config) *AuthService {
	return &AuthService{
		storage:    strg,
		inMemory:   inMemory,
		grpcClient: grpcConn,
		cfg:        cfg,
	}
}

const (
	RegisterCodeKey   = "register_code_"
	ForgotPasswordKey = "forgot_password_code_"
)

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

	go func() {
		err := s.sendVerificationCode(RegisterCodeKey, req.Email)
		if err != nil {
			fmt.Printf("failed to send verification code: %v", err)
		}
	}()

	return &emptypb.Empty{}, nil
}

func (s *AuthService) sendVerificationCode(key, email string) error {
	code, err := utils.GenerateRandomCode(6)
	if err != nil {
		return err
	}

	err = s.inMemory.Set(key+email, code, time.Minute)
	if err != nil {
		return err
	}

	_, err = s.grpcClient.NotificationService().SendEmail(context.Background(), &notification_service.SendEmailRequest{
		To:      email,
		Subject: "Verification email",
		Body: map[string]string{
			"code": code,
		},
		Type: "verification_email",
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *AuthService) Verify(ctx context.Context, req *pb.VerifyRegisterRequest) (*pb.AuthResponse, error) {
	userData, err := s.inMemory.Get("user_" + req.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error: %v", err)
	}

	var user repo.User
	err = json.Unmarshal([]byte(userData), &user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal server error: %v", err)
	}

	code, err := s.inMemory.Get(RegisterCodeKey + user.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Code expired: %v", err)
	}

	if req.Code != code {
		return nil, status.Errorf(codes.Internal, "Incorrect code: %v", err)
	}

	result, err := s.storage.User().Create(&user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Code expired: %v", err)
	}

	token, _, err := utils.CreateToken(s.cfg, &utils.TokenParams{
		UserID:   result.ID,
		Email:    result.Email,
		UserType: result.Type,
		Duration: time.Hour * 24,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal error: %v", err)
	}

	return &pb.AuthResponse{
		Id:          result.ID,
		FirstName:   result.FirstName,
		LastName:    result.LastName,
		Email:       result.Email,
		Type:        result.Type,
		CreatedAt:   result.CreatedAt.Format(time.RFC3339),
		AccessToken: token,
	}, nil
}
