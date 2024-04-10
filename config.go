// 读取 json 配置为 Go 对象

package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

type Config struct {
	// smtp 服务器配置
	Smtp struct {
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Username   string `json:"username"`
		Credential string `json:"credential"`
	} `json:"smtp"`

	// 邮件发送配置
	Mail struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
	} `json:"mail"`

	// 邮件发送时间，cron 表达式
	Cron string `json:"cron"`

	// 智慧房东请求参数
	Zhfd struct {
		RequestUrl    string `json:"requestUrl"`
		Referer       string `json:"referer"`
		Host          string `json:"host"`
		Authorization string `json:"authorization"`
	} `json:"zhfd"`

	// 部署应用的服务器 ip 地址
	ServerIp string `json:"serverIp"`
}

func (c *Config) String() string {
	res, err := json.Marshal(c)
	if err != nil {
		log.Println("json 格式化失败", err)
		return ""
	}
	return string(res)
}

var cache *Config

// ReadConfig 从 config.json 配置文件中读取配置
// 每次读取都会将结果缓存到 cache 中
// 可以通过 useCache 参数决定是否使用缓存的配置
func ReadConfig(useCache bool) (*Config, error) {
	if useCache {
		if cache != nil {
			return cache, nil
		}
		return nil, errors.New("缓存为空")
	}

	bs, err := os.ReadFile("./config.json")
	if err != nil {
		return nil, errors.Join(err, errors.New("读取配置文件失败"))
	}

	cfg := new(Config)
	if err := json.Unmarshal(bs, cfg); err != nil {
		return nil, errors.Join(err, errors.New("读取配置文件失败"))
	}

	cache = cfg

	return cfg, nil
}
