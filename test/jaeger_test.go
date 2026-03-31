package main

import (
	"fmt"
	"testing"

	"github.com/opentracing/opentracing-go"
	// Jaeger 核心配置包
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

// 测试 Jaeger 链路追踪
func TestJaeger(t *testing.T) {
	// 1. 配置 Jaeger
	cfg := config.Configuration{
		// 采样策略：全量采集
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst, // 固定采样
			Param: 1,                       // 1=全采样 0=不采样
		},
		// 上报配置
		Reporter: &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: fmt.Sprintf("http://%s/api/traces", "127.0.0.1:14268"),
		},
	}

	// 2. 初始化全局 Tracer
	closer, err := cfg.InitGlobalTracer(
		"client-test-service", // 服务名（在 Jaeger UI 显示）
		config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		t.Fatalf("Tracer 初始化失败: %v", err)
		return
	}
	defer closer.Close() // 程序结束前关闭

	// 3. 获取全局 Tracer
	tracer := opentracing.GlobalTracer()

	// 4. 创建父 Span（服务入口）
	parentSpan := tracer.StartSpan("A-主任务")
	defer parentSpan.Finish()

	// 5. 调用子函数（传递链路）
	B(tracer, parentSpan)
	C(tracer, parentSpan)
	t.Log("链路追踪上报完成！打开 Jaeger UI 查看")
}

// 子函数 B
func B(tracer opentracing.Tracer, parentSpan opentracing.Span) {
	// 创建子 Span（继承父链路）
	childSpan := tracer.StartSpan(
		"B-子任务",
		opentracing.ChildOf(parentSpan.Context()),
	)
	defer childSpan.Finish()

	// 模拟业务耗时
	fmt.Println("执行 B 任务...")
}

// 子函数 C
func C(tracer opentracing.Tracer, parentSpan opentracing.Span) {
	childSpan := tracer.StartSpan(
		"C-子任务",
		opentracing.ChildOf(parentSpan.Context()),
	)
	defer childSpan.Finish()

	fmt.Println("执行 C 任务...")
}
