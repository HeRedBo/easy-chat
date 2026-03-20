package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/config"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/local/im.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	srv := websocket.NewServer(c.ListenOn,
		websocket.WithServerAuthentication(handler.NewJwtAuth(ctx)),
	)

	defer srv.Stop()

	handler.RegisterHandlers(srv, ctx)

	fmt.Printf("Starting websocket server at %s...\n", c.ListenOn)
	// 3. 监听退出信号（SIGINT/SIGTERM），实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit
		logx.Info("Received exit signal, shutting down server...")
		srv.Stop() // 主动关闭服务器，释放端口
		os.Exit(0)
	}()
	srv.Start()
}
