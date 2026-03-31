package main

import (
	"fmt"
	"time"

	"github.com/HYY-yu/sail-client"
	"github.com/gookit/goutil/dump"
)

type Config struct {
	Name string
	Host string
	Port string
	Mode string

	Database string

	UserRpc struct {
		Etcd struct {
			Hosts []string
			Key   string
		}
	}
	Redisx struct {
		Host string
		Pass string
	}
	JwtAuth struct {
		AccessSecret string
	}
}

func main01() {
	var cfg Config
	s := sail.New(&sail.MetaConfig{
		ETCDEndpoints:  "127.0.0.1:2379",
		ProjectKey:     "98c6f2c2287f4c73cea3d40ae7ec3ff2",
		Namespace:      "user",
		Configs:        "user-api.yaml,user-rpc.yml",
		ConfigFilePath: "./test/conf",
		LogLevel:       "DEBUG",
	}, sail.WithOnConfigChange(func(configFileKey string, s *sail.Sail) {
		if s.Err() != nil {
			fmt.Println(s.Err())
			return
		}

		fmt.Println(s.Pull())

		v, err := s.MergeVipers()
		if err != nil {
			fmt.Println(err)
			return
		}
		v.Unmarshal(&cfg)
		dump.P(&cfg)
		fmt.Println(cfg, "\n", cfg.Database)
	}))
	if s.Err() != nil {
		fmt.Println(s.Err())
		return
	}
	fmt.Println(s.Pull())
	v, err := s.MergeVipers()
	if err != nil {
		fmt.Println(err)
		return
	}
	v.Unmarshal(&cfg)
	fmt.Println(cfg, "\n", cfg.Database)
	for {
		time.Sleep(time.Second)
	}
}
