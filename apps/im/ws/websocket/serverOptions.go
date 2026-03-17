package websocket

import "time"

type ServerOptions func(opt *serverOption)

type serverOption struct {
	Authentication
	patten string

	ack        AckType
	actTimeout time.Duration

	maxConnectionIdle time.Duration
}

func NewServerOption(opts ...ServerOptions) serverOption {

	o := serverOption{
		Authentication: new(authentication),
		actTimeout:     defaultAckTimeout,
		patten:         "/ws",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func WithServerAuthentication(auth Authentication) ServerOptions {
	return func(opt *serverOption) {
		opt.Authentication = auth
	}
}

func WithServerPatten(patten string) ServerOptions {
	return func(opt *serverOption) {
		opt.patten = patten
	}
}

func WithServerAck(ack AckType) ServerOptions {
	return func(opt *serverOption) {
		opt.ack = ack
	}
}

func WithServerMaxConnectionIdle(maxConnectionIdle time.Duration) ServerOptions {
	return func(opt *serverOption) {
		if maxConnectionIdle > 0 {
			opt.maxConnectionIdle = maxConnectionIdle
		}
	}
}
