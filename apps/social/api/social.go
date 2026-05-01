// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/social/api/internal/config"
	"github.com/HeRedBo/easy-chat/apps/social/api/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/social/api/internal/svc"
	"github.com/HeRedBo/easy-chat/pkg/configserver"
	"github.com/HeRedBo/easy-chat/pkg/respx"
	"github.com/HeRedBo/easy-chat/pkg/resultx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/social.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	//conf.MustLoad(*configFile, &c)
	// 使用 GenericConfigService 管理配置
	Configservice := configserver.NewGenericConfigService(*configFile, nil)
	Configservice.SetConfigs("social-api.yaml").
		SetNamespace("social").
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
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	httpx.SetErrorHandlerCtx(resultx.ErrHandler(c.Name))
	httpx.SetOkHandler(resultx.OkHandler)

	// 添加 Request ID 中间件（必须在 Cleanup 之前）
	server.Use(respx.RequestIDMiddleware())
	server.Use(respx.Cleanup())

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
