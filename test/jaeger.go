package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

// 初始化 Jaeger
func initJaeger(serviceName string) io.Closer {
	cfg := config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: "http://127.0.0.1:14268/api/traces",
		},
	}

	closer, err := cfg.InitGlobalTracer(serviceName, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("Jaeger 初始化失败: %v", err))
	}
	return closer
	//defer closer.Close()
}

// HTTP 接口：/test
func testHandler(w http.ResponseWriter, r *http.Request) {
	// 从 HTTP Header 中提取链路（跨服务追踪核心）
	wireContext, _ := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)

	// 创建服务入口 Span
	span := opentracing.StartSpan(
		"http-server-test",
		ext.SpanKindRPCServer, // 标记为服务端
		opentracing.ChildOf(wireContext),
	)
	defer span.Finish()

	// 业务逻辑
	fmt.Println("处理 HTTP 请求")
	queryUser(span)

	w.Write([]byte("链路追踪测试成功！"))
}

// 模拟查询数据库
func queryUser(parentSpan opentracing.Span) {
	childSpan := opentracing.StartSpan(
		"query-user-db",
		opentracing.ChildOf(parentSpan.Context()),
	)
	defer childSpan.Finish()

	fmt.Println("查询用户数据...")
}

func main() {
	// 初始化追踪
	closer := initJaeger("go-http-demo")
	defer closer.Close() // 程序退出是才关闭 
	
	// 启动 HTTP 服务
	http.HandleFunc("/test", testHandler)
	fmt.Println("服务启动：http://127.0.0.1:8095/test")
	http.ListenAndServe(":8095", nil)
}
