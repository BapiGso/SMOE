// Package config 读取 usr/config.yaml：程序自身的设置，与 Hugo 的站点配置（usr/hugo.toml）无关。
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// File 是配置文件的位置，相对于工作目录。
const File = "usr/config.yaml"

type Config struct {
	Name          string `yaml:"name"`     // WebDAV 用户名
	Password      string `yaml:"password"` // WebDAV 密码的 bcrypt 哈希，也用来派生评论审核链接的签名密钥
	BangumiUserID string `yaml:"bangumiUserID"`
	Port          string `yaml:"port"` // HTTP 端口，默认 "95"
	ResendAPI     string `yaml:"resendAPI"`
	MailTo        string `yaml:"mailTo"`
	MailCC        string `yaml:"mailCC"`
}

// Read 读取配置文件。每次调用都重新读，通过 WebDAV 改了密码不用重启。
func Read() (Config, error) {
	b, err := os.ReadFile(File)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
