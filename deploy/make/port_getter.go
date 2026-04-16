package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3" // 推荐使用标准 yaml 库，如果没有需 go get
)

// 定义通用的配置结构体（根据实际 yaml 内容调整）
// 使用 map[string]interface{} 可以兼容 api 和 rpc 不同的结构
type Config map[string]interface{}

func main() {
	// 1. 获取命令行参数: 程序路径 yaml文件路径 模式(api/rpc)
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: getport <yaml_path> <mode>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	mode := os.Args[2]

	// 2. 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// 3. 解析 YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing yaml: %v\n", err)
		os.Exit(1)
	}

	var port string

	// 4. 根据模式提取端口
	switch mode {
	case "api":
		// 获取 Port 字段
		if p, ok := cfg["Port"]; ok {
			// 处理数字类型 (yaml 解析数字默认为 int)
			switch v := p.(type) {
			case int:
				port = strconv.Itoa(v)
			case string:
				port = v
			}
		}
	case "rpc":
		// 获取 ListenOn 字段 (格式通常为 "0.0.0.0:8090")
		if listenOn, ok := cfg["ListenOn"].(string); ok {
			// 按冒号分割，取最后一部分
			parts := strings.Split(listenOn, ":")
			if len(parts) > 0 {
				port = parts[len(parts)-1]
			}
		}
	default:
		fmt.Fprintln(os.Stderr, "Unknown mode:", mode)
		os.Exit(1)
	}

	// 5. 输出纯净的端口号 (不带换行符，方便 Makefile 捕获)
	fmt.Print(port)
}
