package main

import (
	"bufio"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		return // 安静退出，不报错
	}

	path := os.Args[1]
	mode := os.Args[2]

	file, err := os.Open(path)
	if err != nil {
		return // 读不到文件 = 安静退出
	}
	defer file.Close()

	port := "" // 默认空
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// api 和 ws 都从 Port: 字段读取
		if mode == "api" || mode == "ws" {
			if strings.HasPrefix(line, "Port:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					port = parts[1]
				}
				break
			}
		}

		// rpc 从 ListenOn: 读取端口（格式：ListenOn: 0.0.0.0:8080）
		if mode == "rpc" {
			if strings.HasPrefix(line, "ListenOn:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 3 {
					port = strings.TrimSpace(parts[2])
				}
				break
			}
		}

		// mq 或其他无端口类型 → 直接返回空字符串，构建脚本会自动兼容
	}

	print(port) // 输出端口，没有就输出空
}
