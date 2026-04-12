package interceptor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Idempotent interface {
	// 获取请求的标识
	Identify(ctx context.Context, method string) string
	// 是否支持幂等性
	IsIdempotentMethod(fullMethod string) bool
	// 幂等性的验证
	TryAcquire(ctx context.Context, id string) (resp interface{}, isAcquire bool, err error)
	// 执行之后结果的保存
	SaveResp(ctx context.Context, id string, resp interface{}, respErr error) error
}

var (
	// 请求任务标识
	TKey = "easy-chat-idempotence-task-id"
	// 设置rpc调度中rpc请求的标识
	DKey = "easy-chat-idempotence-dispatch-key"

	// Redis 状态
	statusProcessing = "PROCESSING"
	statusSuccess    = "SUCCESS"
	statusFailed     = "FAILED"

	// 锁时间，短一点，防止死锁
	lockTTL = 10 * time.Second
	// 结果缓存时间
	resultTTL = 24 * time.Hour

	defaultIdempotentMethods = map[string]bool{
		"/social.social/GroupCreate": true,
	}
)

// region 客户端与服务端拦截器设置

func ContextWithIdempotentID(ctx context.Context) context.Context {
	// 设置请求的id
	return context.WithValue(ctx, TKey, utils.NewUuid())
}

// NewIdempotenceClient 客户端拦截器：把ID塞入请求头
func NewIdempotenceClient(idempotent Idempotent) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 获取唯一的key
		identify := idempotent.Identify(ctx, method)
		// 在rpc请求中的上下文添加信息
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(DKey, identify))
		// 请求
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// NewIdempotenceServer 服务端拦截器: 幂等核心入口
func NewIdempotenceServer(idempotent Idempotent) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// 获取请求的id
		identify := metadata.ValueFromIncomingContext(ctx, DKey)
		if len(identify) == 0 || !idempotent.IsIdempotentMethod(info.FullMethod) {
			return handler(ctx, req) // 不进行幂等性处理
		}
		key := identify[0]
		// 进入幂等校验
		logx.Infof("【幂等拦截】请求进入 → method: %s, id: %s", info.FullMethod, key)
		r, acquire, err := idempotent.TryAcquire(ctx, key)
		if err != nil {
			logx.Infof("【幂等拦截】重复请求，直接返回结果 → id: %s, err: %v", key, err)
			return nil, err
		}
		// 1 已经有结果 直接返回
		if r != nil {
			fmt.Println("--- 任务已经执行完了 ", identify)
			return r, nil
		}

		// 2. 没有获取到锁 正在执行中
		if !acquire {
			return nil, errors.New("操作正在处理中，请勿重复提交")
			//return nil, errors.Errorf("request is processing, please retry later | key: %s", key)
		}
		// 3. 获取到锁 执行业务
		resp, err = handler(ctx, req)
		fmt.Println("---- 执行任务", key)
		// 保存结果，但是忽略保存错误（不影响业务结果）
		_ = idempotent.SaveResp(ctx, key, resp, err)
		//if err := idempotent.SaveResp(ctx, key, resp, err); err != nil {
		//	return resp, err
		//}
		return resp, err
	}
}

// endregion

// region 定义默认拦截器实现
var (
	DefaultIdempotent       = new(defaultIdempotent)
	DefaultIdempotentClient = NewIdempotenceClient(DefaultIdempotent)
)

type defaultIdempotent struct {
	// 获取和设置请求的id
	*redis.Redis
	// 设置方法对幂等的方法列表
	methods map[string]bool
}

func NewDefaultIdempotent(c redis.RedisConf, methods ...string) Idempotent {
	// 1. 复制默认方法
	m := make(map[string]bool)
	for k, v := range defaultIdempotentMethods {
		m[k] = v
	}
	// 2. 追加自定义方法
	for _, method := range methods {
		m[method] = true
	}
	return &defaultIdempotent{
		Redis:   redis.MustNewRedis(c),
		methods: m,
	}
}

// // 获取请求的标识
// Identify(ctx context.Context, method string) string
func (d *defaultIdempotent) Identify(ctx context.Context, method string) string {
	id, ok := ctx.Value(TKey).(string)
	if !ok || id == "" {
		id = utils.NewUuid()
	}
	// 让其生成请求id
	rpcId := fmt.Sprintf("%v.%s", id, method)
	return rpcId
}

// // 是否支持幂等性
// IsIdempotentMethod(fullMethod string) bool
func (d *defaultIdempotent) IsIdempotentMethod(fullMethod string) bool {
	return d.methods[fullMethod]
}

// // 幂等性的验证
// TryAcquire(ctx context.Context, id string) (resp interface{}, isAcquire bool)
func (d *defaultIdempotent) TryAcquire(ctx context.Context, id string) (resp interface{}, isAcquire bool, err error) {
	// 基于redis实现
	// 1. 查询数据（redis:nil 代表不存在，不是错误）
	resultData, _ := d.Redis.HgetCtx(ctx, id, "data")
	status, _ := d.Redis.HgetCtx(ctx, id, "status")
	// 已经有结果
	if status == statusSuccess || status == statusFailed {
		var resp any
		err = json.Unmarshal([]byte(resultData), &resp)
		if err != nil {
			return nil, false, err
		}
		return resp, false, nil
	}
	// 尝试加锁: 状态 = PROCESSING
	// 使用 hash 保证原子性
	ok, err := d.Redis.SetnxExCtx(ctx, id+":lock", "1", int(lockTTL.Seconds()))
	if err != nil {
		return nil, false, err
	}
	// 拿不到锁 → 正在执行
	if !ok {
		return nil, false, nil
	}

	// 写入处理状态
	err = d.Redis.HmsetCtx(ctx, id, map[string]string{"status": statusProcessing, "time": time.Now().Format(time.RFC3339)})
	if err != nil {
		return nil, false, err
	}
	// 设置过期时间
	_ = d.Redis.ExpireCtx(ctx, id, int(resultTTL.Seconds()))
	return nil, true, nil
}

// SaveResp 执行之后结果的保存
func (d *defaultIdempotent) SaveResp(ctx context.Context, id string, resp interface{}, respErr error) error {
	status := statusSuccess
	if respErr != nil {
		status = statusFailed
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_ = d.Redis.HmsetCtx(ctx, id, map[string]string{
		"status": status,
		"data":   string(data),
		"error":  fmt.Sprintf("%v", respErr),
	})
	_ = d.Redis.ExpireCtx(ctx, id, int(resultTTL))
	return nil
}

// endregion
