package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/config"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/server"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/user"
	"github.com/HeRedBo/easy-chat/pkg/configserver"
	"github.com/HeRedBo/easy-chat/pkg/env"
	"github.com/HeRedBo/easy-chat/pkg/interceptor/rpcserver"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	//conf.MustLoad(*configFile, &c)

	// 使用 GenericConfigService 管理配置
	var baseConfig *configserver.Config
	var etcFileName = "user-rpc.yaml"
	if env.IsDockerContainer() {
		// 修复 Docker 容器中 ETCD 地址问题
		baseConfig = &configserver.Config{
			ETCDEndpoints:  "host.docker.internal:2379",
			ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
			ConfigFilePath: "", // 空字符串表示不存储本地配置文件
			LogLevel:       "DEBUG",
		}
		etcFileName = "user-rpc-dev.yaml"
	}

	Configservice := configserver.NewGenericConfigService(*configFile, baseConfig)
	Configservice.SetConfigs(etcFileName).
		SetNamespace("user").
		SetRunFunc(func(v any) {
			// 类型断言
			cfg, ok := v.(*config.Config)
			if !ok {
				panic(fmt.Sprintf("invalid config type: %T", v)) // 直接 panic 终止程序
			}
			Run(*cfg)
		})

	// 启动配置服务
	if err := Configservice.Start(&c); err != nil {
		panic(err)
	}
	// 等待服务完成
	Configservice.Wait()

}

func Run(c config.Config) {
	ctx := svc.NewServiceContext(c)

	if err := ctx.SetRootToken(); err != nil {
		panic(err)
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		user.RegisterUserServer(grpcServer, server.NewUserServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(rpcserver.LogInterceptor)
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	// 3. 监听退出信号（SIGINT/SIGTERM），实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit
		logx.Info("Received exit signal, shutting down server...")
		s.Stop() // 主动关闭服务器，释放端口
		os.Exit(0)
	}()
	s.Start()
}
