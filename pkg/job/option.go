package job

import "time"

type (
	RetryOptions func(opts *retryOptions)

	retryOptions struct {
		timeout     time.Duration
		retryNums   int
		IsRetryFunc IsRetryFunc
		RetryJetLag RetryJetLagFunc
	}
)

func newOptions(opts ...RetryOptions) *retryOptions {
	opt := &retryOptions{
		timeout:     DefaultRetryTimeout,
		retryNums:   DefaultRetryNums,
		IsRetryFunc: RetryAlways,
		RetryJetLag: RetryJetLagAlways,
	}
	for _, optios := range opts {
		optios(opt)
	}
	return opt
}

func WithRetryTimeout(timeout time.Duration) RetryOptions {
	return func(opts *retryOptions) {
		if timeout > 0 {
			opts.timeout = timeout
		}
	}
}

func WithRetryNums(nums int) RetryOptions {
	return func(opts *retryOptions) {
		opts.retryNums = 1
		if nums > 1 {
			opts.retryNums = nums
		}
	}
}

func WithRetryIsFunc(retryFunc IsRetryFunc) RetryOptions {
	return func(opts *retryOptions) {
		if retryFunc != nil {
			opts.IsRetryFunc = retryFunc
		}
	}
}

func WithRetryJetLagFunc(jetLagFunc RetryJetLagFunc) RetryOptions {
	return func(opts *retryOptions) {
		if jetLagFunc != nil {
			opts.RetryJetLag = jetLagFunc
		}
	}
}
