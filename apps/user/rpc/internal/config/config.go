package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql  sqlx.SqlConf
	Cache  cache.CacheConf
	Redisx redis.RedisConf
	Jwt    struct {
		AccessSecret string
		AccessExpire int64
	}
}
