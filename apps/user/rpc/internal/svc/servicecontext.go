package svc

import (
	"github.com/HeRedBo/easy-chat/apps/user/models"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/internal/config"
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
	// 2. 初始化用户模型
	userModel := models.NewUsersModel(sqlConn, c.Cache)
	return &ServiceContext{
		Config:     c,
		Redis:      redis.MustNewRedis(c.Redisx),
		UsersModel: userModel, // 注入到上下文
	}
}
