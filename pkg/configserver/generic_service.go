package configserver

import (
	"sync"

	"github.com/zeromicro/go-zero/core/proc"
)

// 默认配置
var defaultConfig = &Config{
	ETCDEndpoints:  "127.0.0.1:2379",
	ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
	ConfigFilePath: "", // 空字符串表示不存储本地配置文件
	LogLevel:       "DEBUG",
}

// GenericConfigService 通用配置服务
type GenericConfigService struct {
	configFile string
	config     *Config
	server     *configServer
	wg         sync.WaitGroup
	runFunc    func(any)
}

// NewGenericConfigService 创建通用配置服务实例
func NewGenericConfigService(configFile string, baseConfig *Config) *GenericConfigService {
	// 如果没有传递配置，使用默认配置
	if baseConfig == nil {
		baseConfig = defaultConfig
	}

	return &GenericConfigService{
		configFile: configFile,
		config:     baseConfig,
	}
}

// SetConfigs 设置配置文件名称
func (s *GenericConfigService) SetConfigs(configs string) *GenericConfigService {
	s.config.Configs = configs
	return s
}

// SetNamespace 设置命名空间
func (s *GenericConfigService) SetNamespace(namespace string) *GenericConfigService {
	s.config.Namespace = namespace
	return s
}

// SetLogLevel 设置日志级别
func (s *GenericConfigService) SetLogLevel(logLevel string) *GenericConfigService {
	s.config.LogLevel = logLevel
	return s
}

// SetRunFunc 设置运行函数
func (s *GenericConfigService) SetRunFunc(runFunc func(any)) *GenericConfigService {
	s.runFunc = runFunc
	return s
}

// Start 启动配置服务
func (s *GenericConfigService) Start(v any) error {
	// 初始化配置服务器
	sail := NewSail(s.config)
	s.server = NewConfigServer(s.configFile, sail)

	// 定义配置变更回调
	onChange := func(data []byte) error {
		// 清理旧资源
		proc.WrapUp()

		// 解析新配置
		if err := LoadFromJsonBytes(data, v); err != nil {
			return err
		}

		// 启动新实例
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if s.runFunc != nil {
				s.runFunc(v)
			}
		}()

		return nil
	}

	// 加载配置并启动
	if err := s.server.MustLoad(v, onChange); err != nil {
		return err
	}

	// 启动第一个实例
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if s.runFunc != nil {
			s.runFunc(v)
		}
	}()

	return nil
}

// Wait 等待所有运行实例完成
func (s *GenericConfigService) Wait() {
	s.wg.Wait()
}

// Stop 停止配置服务
func (s *GenericConfigService) Stop() {
	proc.WrapUp()
}
