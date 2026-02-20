package tools

import (
	"crypto/tls"
	"net/smtp"
)

type Email struct {
	user   string
	passwd string
	Auth   smtp.Auth
	Conn   *tls.Conn
}
