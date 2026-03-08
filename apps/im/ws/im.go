package main

import (
	"flag"
	"fmt"

	handler "github.com/HeRedBo/easy-chat/apps/im/ws/internal"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/config"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/local/im.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	srv := websocket.NewServer(c.ListenOn)
	defer srv.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(srv, ctx)
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	srv.Start()
}
