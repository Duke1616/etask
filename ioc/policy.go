package ioc

import (
	endpointv1 "github.com/Duke1616/etask/api/proto/gen/ecmdb/endpoint/v1"
	policyv1 "github.com/Duke1616/etask/api/proto/gen/ecmdb/policy/v1"
	grpcpkg "github.com/Duke1616/etask/pkg/grpc"
	"github.com/Duke1616/etask/pkg/grpc/registry"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// InitECMDBGrpcClient 初始化 ECMDB gRPC 客户端
func InitECMDBGrpcClient(reg registry.Registry) grpc.ClientConnInterface {
	var cfg grpcpkg.ClientConfig
	if err := viper.UnmarshalKey("grpc.client.ecmdb", &cfg); err != nil {
		panic(err)
	}

	cc, err := grpcpkg.NewClientConn(
		reg,
		grpcpkg.WithServiceName(cfg.Name),
		grpcpkg.WithClientJWTAuth(cfg.AuthToken),
	)
	if err != nil {
		panic(err)
	}

	return cc
}

// InitPolicyServiceClient 初始化 Policy 服务客户端
func InitPolicyServiceClient(cc grpc.ClientConnInterface) policyv1.PolicyServiceClient {
	return policyv1.NewPolicyServiceClient(cc)
}

// InitEndpointServiceClient 初始化 Endpoint 服务客户端
func InitEndpointServiceClient(cc grpc.ClientConnInterface) endpointv1.EndpointServiceClient {
	return endpointv1.NewEndpointServiceClient(cc)
}
