// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/HeRedBo/easy-chat/apps/im/rpc/imclient"
	"github.com/HeRedBo/easy-chat/apps/social/api/internal/config"
	"github.com/HeRedBo/easy-chat/apps/social/rpc/socialclient"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/userclient"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	userclient.User
	socialclient.Social
	imclient.Im
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Social: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc)),
		Im:     imclient.NewIm(zrpc.MustNewClient(c.Imrpc)),
	}
}
