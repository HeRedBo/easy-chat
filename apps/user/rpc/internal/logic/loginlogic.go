package logic

import (
	"context"
	"time"

	"github.com/HeRedBo/easy-chat/apps/user/models"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/user"
	"github.com/HeRedBo/easy-chat/pkg/ctxdata"
	"github.com/HeRedBo/easy-chat/pkg/encrypt"
	"github.com/HeRedBo/easy-chat/pkg/xerr"
	"github.com/gookit/goutil/dump"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	//ErrPhoneNotRegister = errors.New("手机号没有注册")
	//ErrUserPwdError     = errors.New("密码不正确")
	ErrPhoneNotRegister = xerr.New(xerr.SERVER_COMMON_ERROR, "手机号码没有注册")
	ErrUserPwdError     = xerr.New(xerr.SERVER_COMMON_ERROR, "密码不正确")
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
			//return nil, ErrPhoneNotRegister
			return nil, errors.WithStack(ErrPhoneNotRegister)
		}
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by phone err %v , req %v", err, in.Phone)
	}
	// 密码验证
	if !encrypt.ValidatePasswordHash(in.Password, userEntity.Password.String) {
		return nil, errors.WithStack(ErrUserPwdError)
		//return nil, ErrUserPwdError
	}
	dump.P("结束")
	return nil, errors.New("做测试")
	// 生成token
	now := time.Now().Unix()
	token, err := ctxdata.GetJwtToken(l.svcCtx.Config.Jwt.AccessSecret, now, l.svcCtx.Config.Jwt.AccessExpire,
		userEntity.Id)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "ctxdata get jwt token err %v", err)
		//return nil, err
	}
	var u user.UserEntity
	_ = copier.Copy(&u, userEntity)
	return &user.LoginResp{
		Token:  token,
		Id:     userEntity.Id,
		User:   &u,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
