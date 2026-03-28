// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/user/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/user/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/userclient"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户登入
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// todo: add your logic here and delete this line
	loginRes, err := l.svcCtx.User.Login(l.ctx, &userclient.LoginReq{
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	var res types.LoginResp
	copier.Copy(&res, loginRes)

	// 设置用户在线
	_ = l.svcCtx.Redis.HsetCtx(l.ctx, constants.REDIS_ONLINE_USER, loginRes.Id, "1")
	
	return &res, nil
}
