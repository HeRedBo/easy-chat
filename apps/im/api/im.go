// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/im/api/internal/config"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/svc"
	"github.com/HeRedBo/easy-chat/pkg/respx"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/im.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)
	// 统一返回格式
	respx.Register()
	server.Use(respx.Cleanup())

	// 3. 监听退出信号（SIGINT/SIGTERM），实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit
		logx.Info("Received exit signal, shutting down server...")
		server.Stop() // 主动关闭服务器，释放端口
		os.Exit(0)
	}()
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
