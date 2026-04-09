package resp

// HTTP 状态码语义
// 200 成功
// 400 参数错误
// 401 未登录/Token过期
// 403 无权限
// 404 资源不存在
// 429 请求限流
// 500 服务器错误
// 504 超时

// 业务错误码
const (
	Success        = 200
	ParamError     = 400
	Unauthorized   = 401
	Forbidden      = 403
	NotFound       = 404
	RateLimit      = 429
	ServerError    = 500
	GatewayTimeout = 504
)
