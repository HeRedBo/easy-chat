// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/user/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/user/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/user"
	"github.com/HeRedBo/easy-chat/pkg/ctxdata"
	"github.com/HeRedBo/easy-chat/pkg/respx"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户信息
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	// todo: add your logic here and delete this line
	//return nil, xerr.NewAuthErr("授权自定义业务错误信息")
	uid := ctxdata.GetUid(l.ctx)
	userInfoResp, err := l.svcCtx.GetUserInfo(l.ctx, &user.GetUserInfoReq{
		Id: uid,
	})
	if err != nil {
		return nil, err
	}
	var res types.User
	copier.Copy(&res, userInfoResp.User)
	respx.SetSuccessMsg(l.ctx, "获取用户信息成功")
	return &types.UserInfoResp{Info: res}, nil
}
