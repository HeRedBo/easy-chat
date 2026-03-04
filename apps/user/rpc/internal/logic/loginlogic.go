package logic

import (
	"context"
	"errors"
	"time"

	"github.com/HeRedBo/easy-chat/apps/user/models"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/user"
	"github.com/HeRedBo/easy-chat/pkg/ctxdata"
	"github.com/HeRedBo/easy-chat/pkg/encrypt"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrPhoneNotRegister = errors.New("手机号没有注册")
	ErrUserPwdError     = errors.New("密码不正确")
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginResp, error) {
	// todo: add your logic here and delete this line
	// 1. 验证用户是否注册，根据手机号码验证
	userEntity, err := l.svcCtx.UsersModel.FindByPhone(l.ctx, in.Phone)
	if err != nil {
		if err == models.ErrNotFound {
			return nil, ErrPhoneNotRegister
		}
		return nil, err
	}
	// 密码验证
	if !encrypt.ValidatePasswordHash(in.Password, userEntity.Password.String) {
		return nil, ErrUserPwdError
	}
	// 生成token
	now := time.Now().Unix()
	token, err := ctxdata.GetJwtToken(l.svcCtx.Config.Jwt.AccessSecret, now, l.svcCtx.Config.Jwt.AccessExpire,
		userEntity.Id)
	if err != nil {
		return nil, err
	}
	return &user.LoginResp{
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
