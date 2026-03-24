package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/config"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/task.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	if err := c.SetUp(); err != nil {
		panic(err)
	}
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	ctx := svc.NewServiceContext(c)
	listen := handler.NewListen(ctx)

	for _, s := range listen.Services() {
		serviceGroup.Add(s)
	}
	fmt.Printf("Starting mqueue server")
	// 3. 监听退出信号（SIGINT/SIGTERM），实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit
		logx.Info("Received exit signal, shutting down server...")
		serviceGroup.Stop() // 主动关闭服务器，释放端口
		os.Exit(0)
	}()
	serviceGroup.Start()
}
