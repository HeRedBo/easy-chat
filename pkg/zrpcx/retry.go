package zrpcx

import (
	"encoding/json"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// RetryPolicy defines the retry policy for a gRPC service.
type RetryPolicy struct {
	MaxAttempts          int      `json:"MaxAttempts"`
	InitialBackoff       string   `json:"InitialBackoff"`
	MaxBackoff           string   `json:"MaxBackoff"`
	BackoffMultiplier    float64  `json:"BackoffMultiplier"`
	RetryableStatusCodes []string `json:"RetryableStatusCodes"`
}

type serviceConfig struct {
	LoadBalancingPolicy string         `json:"loadBalancingPolicy"`
	MethodConfig        []methodConfig `json:"methodConfig"`
}

type methodConfig struct {
	Name         []nameConfig `json:"name"`
	WaitForReady bool         `json:"waitForReady"`
	RetryPolicy  RetryPolicy  `json:"retryPolicy"`
}

type nameConfig struct {
	Service string `json:"service"`
}

// BuildGlobalRetryOption merges all retry policies into a single gRPC ServiceConfig.
// This option can be shared across all zrpc.MustNewClient calls within the same service.
// It explicitly preserves go-zero's p2c load balancing policy.
func BuildGlobalRetryOption(policies map[string]RetryPolicy) zrpc.ClientOption {
	cfg := serviceConfig{
		LoadBalancingPolicy: "p2c",
	}

	if len(policies) > 0 {
		mc := make([]methodConfig, 0, len(policies))
		for svcName, rp := range policies {
			mc = append(mc, methodConfig{
				Name:         []nameConfig{{Service: svcName}},
				WaitForReady: true,
				RetryPolicy:  rp,
			})
		}
		cfg.MethodConfig = mc
	}

	raw, _ := json.Marshal(cfg)
	return zrpc.WithDialOption(grpc.WithDefaultServiceConfig(string(raw)))
}
