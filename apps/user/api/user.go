// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/user/api/internal/config"
	"github.com/HeRedBo/easy-chat/apps/user/api/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/user/api/internal/svc"
	"github.com/HeRedBo/easy-chat/pkg/configserver"
	"github.com/HeRedBo/easy-chat/pkg/env"
	"github.com/HeRedBo/easy-chat/pkg/respx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

var wg sync.WaitGroup

func main() {
	flag.Parse()

	var c config.Config
	//conf.MustLoad(*configFile, &c)
	var baseConfig *configserver.Config
	if env.IsDockerContainer() {
		// 修复 Docker 容器中 ETCD 地址问题
		baseConfig = &configserver.Config{
			ETCDEndpoints:  "host.docker.internal:2379",
			ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
			Namespace:      "user",
			Configs:        "user-api-dev.yaml",
			ConfigFilePath: "", // 空字符串表示不存储本地配置文件
			LogLevel:       "DEBUG",
		}
	} else {
		baseConfig = &configserver.Config{
			ETCDEndpoints: "127.0.0.1:2379",
			ProjectKey:    "98c6f2c2287f4c73cea3d40ae7ec3ff2",
			Namespace:     "user",
			Configs:       "user-api.yaml",
			//ConfigFilePath: "./etc/user.yaml",
			ConfigFilePath: "", // 空字符串代表不存储本地配置文件
			LogLevel:       "DEBUG",
		}
	}

	err := configserver.NewConfigServer(*configFile, configserver.NewSail(baseConfig)).MustLoad(&c, func(bytes []byte) error {
		var c config.Config
		_ = configserver.LoadFromJsonBytes(bytes, &c)
		fmt.Println("配置更新", c)
		// 配置更新了！
		proc.WrapUp() // 优雅关闭旧资源
		// 启新服务
		wg.Add(1)
		go func(c config.Config) {
			defer wg.Done()
			Run(c)
		}(c)
		return nil
	})

	if err != nil {
		panic(err)
	}
	wg.Add(1)
	go func(c config.Config) {
		defer wg.Done()
		Run(c)
	}(c)

	// 等待所有 goroutine 完成
	wg.Wait()
}

func Run(c config.Config) {
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 初始化全局上下文（DB、Redis、RPC 等）
	ctx := svc.NewServiceContext(c)
	// 注册所有 API 接口
	handler.RegisterHandlers(server, ctx)

	// 统一返回格式
	respx.Register()
	// 添加 Request ID 中间件（必须在 Cleanup 之前）
	server.Use(respx.RequestIDMiddleware())
	server.Use(respx.Cleanup())
	//httpx.SetErrorHandlerCtx(resultx.ErrHandler(c.Name))
	// httpx.SetOkHandler(resultx.OkHandler)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)

	// 3. 监听退出信号（SIGINT/SIGTERM），实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit
		logx.Info("Received exit signal, shutting down server...")
		server.Stop() // 主动关闭服务器，释放端口
		os.Exit(0)
	}()
	server.Start()
}
