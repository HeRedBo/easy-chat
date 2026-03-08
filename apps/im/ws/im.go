package main

import (
	"flag"
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/config"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/handler"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/zeromicro/go-zero/core/conf"
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

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	srv.Start()
}
