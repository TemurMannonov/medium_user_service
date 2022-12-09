package grpc_client

import (
	"fmt"

	"github.com/TemurMannonov/medium_api_gateway/config"
	pbu "github.com/TemurMannonov/medium_api_gateway/genproto/user_service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcClientI interface {
	UserService() pbu.UserServiceClient
	AuthService() pbu.AuthServiceClient
}

type GrpcClient struct {
	cfg         config.Config
	connections map[string]interface{}
}

func New(cfg config.Config) (GrpcClientI, error) {
	connUserService, err := grpc.Dial(
		fmt.Sprintf("%s%s", cfg.UserServiceHost, cfg.UserServiceGrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("user service dial host: %s port:%s err: %v",
			cfg.UserServiceHost, cfg.UserServiceGrpcPort, err)
	}

	return &GrpcClient{
		cfg: cfg,
		connections: map[string]interface{}{
			"user_service": pbu.NewUserServiceClient(connUserService),
			"auth_service": pbu.NewAuthServiceClient(connUserService),
		},
	}, nil
}

func (g *GrpcClient) UserService() pbu.UserServiceClient {
	return g.connections["user_service"].(pbu.UserServiceClient)
}

func (g *GrpcClient) AuthService() pbu.AuthServiceClient {
	return g.connections["auth_service"].(pbu.AuthServiceClient)
}
