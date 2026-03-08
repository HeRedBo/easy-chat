package mq

import (
	"flag"
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/config"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
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
	fmt.Println("Starting mqueue server at ...")

	serviceGroup.Start()
}
