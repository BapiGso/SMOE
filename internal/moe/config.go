package moe

import (
	"os"

	"gopkg.in/yaml.v3"
)

// configFile 是程序自身的设置，与 Hugo 的站点配置（usr/hugo.toml）无关。
const configFile = "usr/config.yaml"

type config struct {
	Name          string `yaml:"name"`     // WebDAV 用户名
	Password      string `yaml:"password"` // WebDAV 密码的 bcrypt 哈希，也用来派生评论审核链接的签名密钥
	BangumiUserID string `yaml:"bangumiUserID"`
	Port          string `yaml:"port"` // HTTP 端口，默认 "95"
	ResendAPI     string `yaml:"resendAPI"`
	MailTo        string `yaml:"mailTo"`
	MailCC        string `yaml:"mailCC"`
}

// readConfig 读取配置文件。WebDAV 认证每次都重新读，通过 WebDAV 改了密码不用重启。
func readConfig() (config, error) {
	var cfg config
	b, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(b, &cfg)
	return cfg, err
}
