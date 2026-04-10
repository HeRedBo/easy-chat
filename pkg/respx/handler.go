package respx

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/HeRedBo/easy-chat/pkg/storex"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	xerrors "github.com/zeromicro/x/errors" // 改名，避免冲突
)

// 全局唯一 key，防止冲突
type successMsgKey struct{}

// SetSuccessMsg 存储当前请求的成功消息
func SetSuccessMsg(msg string) {
	storex.Set(successMsgKey{}, msg)
}

// GetSuccessMsg 获取当前请求的成功消息
func GetSuccessMsg() string {
	v, ok := storex.Get(successMsgKey{})
	if !ok {
		return "success"
	}
	msg, ok := v.(string)
	if !ok {
		return "success"
	}
	return msg
}

// Cleanup 中间件：请求结束自动清理
func Cleanup() rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer storex.Clean()
			next(w, r)
		}
	}
}

// Register 注册全局处理器
func Register() {
	// 成功返回
	httpx.SetOkHandler(OkHandler)
	// 错误返回
	httpx.SetErrorHandlerCtx(ErrHandler)
}

// OkHandler 成功包装
func OkHandler(ctx context.Context, data interface{}) any {
	return Ok(ctx, data)
}

// 全局错误处理（自动识别所有错误）
func ErrHandler(ctx context.Context, err error) (int, any) {
	logx.WithContext(ctx).Infof("ErrHandler called: %v", err) // 新增
	var (
		httpCode = http.StatusInternalServerError // 默认500
		code     = ServerError
		msg      = "服务器异常"
	)

	// 1. 处理自定义 CodeMsg 错误（业务主动抛出）
	var cm *xerrors.CodeMsg
	errStr := err.Error()
	if errors.As(err, &cm) {
		code = cm.Code
		msg = cm.Msg
		httpCode = codeToHttp(code)
		goto log
	}

	// 2. 处理 grpc 错误
	if st, ok := status.FromError(err); ok {
		msg = st.Message()
		switch st.Code() {
		case codes.InvalidArgument:
			httpCode = http.StatusBadRequest
			code = ParamError
		case codes.PermissionDenied:
			httpCode = http.StatusForbidden
			code = Forbidden
		case codes.Unauthenticated:
			httpCode = http.StatusUnauthorized
			code = Unauthorized
		case codes.NotFound:
			httpCode = http.StatusNotFound
			code = NotFound
		case codes.DeadlineExceeded:
			httpCode = http.StatusGatewayTimeout
			code = GatewayTimeout
		default:
			httpCode = http.StatusInternalServerError
			code = ServerError
		}
		goto log
	}

	// 3. 处理 go-zero 内置错误（JWT、参数、限流、超时）
	switch {
	case containsAny(errStr, "token", "expired", "invalid"):
		httpCode = http.StatusUnauthorized
		code = Unauthorized
		msg = "登录已过期"

	case containsAny(errStr, "validation", "unmarshal", "parse"):
		httpCode = http.StatusBadRequest
		code = ParamError
		msg = "参数错误"

	case containsAny(errStr, "rate limit", "too many"):
		httpCode = http.StatusTooManyRequests
		code = RateLimit
		msg = "请求频繁"

	case containsAny(errStr, "deadline", "timeout"):
		httpCode = http.StatusGatewayTimeout
		code = GatewayTimeout
		msg = "请求超时"

	case containsAny(errStr, "not found"):
		httpCode = http.StatusNotFound
		code = NotFound
		msg = "资源不存在"
	}

log:
	// 日志记录
	logx.WithContext(ctx).Errorf("err: %s", err.Error())
	logx.WithContext(ctx).Errorf("final response: code=%d http=%d msg=%s", code, httpCode, msg)
	return httpCode, Fail(code, msg)
}

// 业务码 → HTTP 状态映射
func codeToHttp(code int) int {
	switch code {
	case ParamError:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	case RateLimit:
		return http.StatusTooManyRequests
	case ServerError, GatewayTimeout:
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}

// 字符串包含判断
func containsAny(s string, items ...string) bool {
	for _, v := range items {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}
