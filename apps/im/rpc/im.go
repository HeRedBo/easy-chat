package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/im/rpc/im"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/internal/config"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/internal/server"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/internal/svc"
	"github.com/HeRedBo/easy-chat/pkg/configserver"
	"github.com/HeRedBo/easy-chat/pkg/interceptor/rpcserver"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/im.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	//conf.MustLoad(*configFile, &c)
	// 使用 GenericConfigService 管理配置
	Configservice := configserver.NewGenericConfigService(*configFile, nil)
	Configservice.SetConfigs("im-rpc.yaml").
		SetNamespace("im").
		SetRunFunc(func(v any) {
			// 类型断言
			cfg, ok := v.(*config.Config)
			if !ok {
				panic(fmt.Sprintf("配置类型错误，期望 *config.Config，实际得到：%T", v))
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

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		im.RegisterImServer(grpcServer, server.NewImServer(ctx))

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
