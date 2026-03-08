package svc

import (
	"time"

	"github.com/HeRedBo/easy-chat/apps/user/models"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/config"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/HeRedBo/easy-chat/pkg/ctxdata"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config
	*redis.Redis
	models.UsersModel // 数据库模型实例
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 初始化MySQL连接（go-zero内置连接池管理）
	sqlConn := sqlx.NewMysql(c.Mysql.DataSource)
	return &ServiceContext{
		Config:     c,
		Redis:      redis.MustNewRedis(c.Redisx),
		UsersModel: models.NewUsersModel(sqlConn, c.Cache), // 初始化用户模型 注入到上下文
	}
}

func (svc *ServiceContext) SetRootToken() error {
	// 生成jwt
	systemToken, err := ctxdata.GetJwtToken(svc.Config.Jwt.AccessSecret, time.Now().Unix(), 999999999, constants.SYSTEM_ROOT_UID)
	if err != nil {
		return err
	}
	// 写入到redis
	return svc.Redis.Set(constants.REDIS_SYSTEM_ROOT_TOKEN, systemToken)
}
