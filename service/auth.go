package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TemurMannonov/medium_user_service/config"
	"github.com/TemurMannonov/medium_user_service/genproto/notification_service"
	pb "github.com/TemurMannonov/medium_user_service/genproto/user_service"
	"github.com/TemurMannonov/medium_user_service/pkg/utils"
	"github.com/TemurMannonov/medium_user_service/storage/repo"
	"github.com/sirupsen/logrus"
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
	logger     *logrus.Logger
}

func NewAuthService(strg storage.StorageI, inMemory storage.InMemoryStorageI, grpcConn grpcPkg.GrpcClientI, cfg *config.Config, logger *logrus.Logger) *AuthService {
	return &AuthService{
		storage:    strg,
		inMemory:   inMemory,
		grpcClient: grpcConn,
		cfg:        cfg,
		logger:     logger,
	}
}

const (
	RegisterCodeKey   = "register_code_"
	ForgotPasswordKey = "forgot_password_code_"
)

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*emptypb.Empty, error) {
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to marshal: %v", err)
	}

	err = s.inMemory.Set("user_"+user.Email, string(userData), 10*time.Minute)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set to rd: %v", err)
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
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	var user repo.User
	err = json.Unmarshal([]byte(userData), &user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal: %v", err)
	}

	code, err := s.inMemory.Get(RegisterCodeKey + user.Email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "code_expired")
	}

	if req.Code != code {
		return nil, status.Errorf(codes.Internal, "incorrect_code")
	}

	result, err := s.storage.User().Create(&user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	token, _, err := utils.CreateToken(s.cfg, &utils.TokenParams{
		UserID:   result.ID,
		Email:    result.Email,
		UserType: result.Type,
		Duration: time.Hour * 24,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
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

func (s *AuthService) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.AuthPayload, error) {
	accessToken := req.AccessToken

	payload, err := utils.VerifyToken(s.cfg, accessToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	hasPermission, err := s.storage.Permission().CheckPermission(payload.UserType, req.Resource, req.Action)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	return &pb.AuthPayload{
		Id:            payload.ID.String(),
		UserId:        payload.UserID,
		Email:         payload.Email,
		UserType:      payload.UserType,
		IssuedAt:      payload.IssuedAt.Format(time.RFC3339),
		ExpiredAt:     payload.ExpiredAt.Format(time.RFC3339),
		HasPermission: hasPermission,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	user, err := s.storage.User().GetByEmail(req.Email)
	if err != nil {
		s.logger.WithError(err).Error("failed to get user by email")
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}

	err = utils.CheckPassword(req.Password, user.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "incorrect_password")
	}

	token, _, err := utils.CreateToken(s.cfg, &utils.TokenParams{
		UserID:   user.ID,
		Email:    user.Email,
		UserType: user.Type,
		Duration: time.Hour * 24,
	})
	if err != nil {
		s.logger.WithError(err).Error("failed to create token")
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	return &pb.AuthResponse{
		Id:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Type:        user.Type,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		AccessToken: token,
	}, nil
}
