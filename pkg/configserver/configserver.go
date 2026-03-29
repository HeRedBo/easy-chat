package configserver

import (
	"errors"

	"github.com/zeromicro/go-zero/core/conf"
)

var ErrNotSetConfig = errors.New("configserver: not set config")

type OnChange func([]byte) error

type ConfigServer interface {
	Build() error
	SetOnChange(OnChange OnChange)
	FormJsonBytes() ([]byte, error)
}

type configServer struct {
	ConfigServer
	configFIle string
}

func NewConfigServer(configFIle string, p ConfigServer) *configServer {
	return &configServer{
		ConfigServer: p,
		configFIle:   configFIle,
	}
}

func (s *configServer) MustLoad(v any, onChange OnChange) error {
	if s.configFIle == "" && s.ConfigServer == nil {
		return ErrNotSetConfig
	}

	if s.ConfigServer == nil {
		// 使用go-zero的默认
		conf.MustLoad(s.configFIle, v)
		return nil
	}

	if onChange != nil {
		s.SetOnChange(onChange)
	}

	if err := s.ConfigServer.Build(); err != nil {
		return err
	}

	data, err := s.ConfigServer.FormJsonBytes()
	if err != nil {
		return err
	}
	return LoadFromJsonBytes(data, v)
}

func LoadFromJsonBytes(data []byte, v any) error {
	return conf.LoadFromJsonBytes(data, v)
}
