package xerr

import (
	"net/http"
)

// BizError 业务自定义错误：自带 业务码 + 消息 + HTTP状态码
type BizError struct {
	Code     int
	Msg      string
	HttpCode int
	Cause    error
}

// 实现 error 接口
func (e *BizError) Error() string {
	if e.Cause != nil {
		return e.Msg + ": " + e.Cause.Error()
	}
	return e.Msg
}

func (e *BizError) Unwrap() error {
	return e.Cause
}

// 基础创建
func New(code int, msg string) error {
	return &BizError{
		Code:     code,
		Msg:      msg,
		HttpCode: http.StatusInternalServerError,
	}
}

// 创建：带 HTTP 码（内部使用）
func NewWithHttp(code int, httpCode int, msg string) error {
	return &BizError{
		Code:     code,
		Msg:      msg,
		HttpCode: httpCode,
	}
}

func NewMsg(msg ...string) error {
	defaultMsg := ErrMsg(SERVER_COMMON_ERROR)
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(SERVER_COMMON_ERROR, http.StatusInternalServerError, defaultMsg)
}

func NewReqParamErr(msg ...string) error {
	defaultMsg := ErrMsg(REQUEST_PARAM_ERROR)
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(REQUEST_PARAM_ERROR, http.StatusBadRequest, defaultMsg)
}

func NewDBErr(msg ...string) error {
	defaultMsg := ErrMsg(DB_ERROR)
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(DB_ERROR, http.StatusInternalServerError, defaultMsg)
}

func NewInternalErr(msg ...string) error {
	defaultMsg := ErrMsg(SERVER_COMMON_ERROR)
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(SERVER_COMMON_ERROR, http.StatusInternalServerError, defaultMsg)
}

func NewTokenExpireErr(msg ...string) error {
	defaultMsg := ErrMsg(TOKEN_EXPIRE_ERROR)
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(TOKEN_EXPIRE_ERROR, http.StatusUnauthorized, defaultMsg)
}

// NewSysErr 系统错误（同 NewInternalErr）
func NewSysErr(msg ...string) error {
	return NewInternalErr(msg...)
}

// NewAuthErr 未登录 / 授权失败
func NewAuthErr(msg ...string) error {
	defaultMsg := "请先登录"
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(TOKEN_EXPIRE_ERROR, http.StatusUnauthorized, defaultMsg)
}

// NewForbiddenErr 无权限访问
func NewForbiddenErr(msg ...string) error {
	defaultMsg := "无权限访问"
	if len(msg) > 0 && msg[0] != "" {
		defaultMsg = msg[0]
	}
	return NewWithHttp(403, http.StatusForbidden, defaultMsg)
}

// NewHttpError 完全自定义：业务码 + HTTP码 + 消息
func NewHttpError(code int, httpCode int, msg string) error {
	return NewWithHttp(code, httpCode, msg)
}
