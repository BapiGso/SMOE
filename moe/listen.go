package moe

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/acme/autocert"
)

func (s *Smoe) Listen() {
	srv := s.cfg.Server
	port := srv.Port
	if port == "" {
		port = "80"
	}

	// Let's Encrypt：domain 非空，固定用 :80/:443，忽略 port/httpsPort
	if srv.Domain != "" {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache("usr/.autocert"),
			HostPolicy: autocert.HostWhitelist(srv.Domain),
		}
		go http.ListenAndServe(":80", m.HTTPHandler(nil))
		fmt.Printf(banner, "=> https://"+srv.Domain)
		log.Fatal(http.Serve(m.Listener(), s.e))
		return
	}

	// 自签名证书：httpsPort 非空时并行启动
	if srv.HttpsPort != "" {
		go func() {
			sc := echo.StartConfig{Address: ":" + srv.HttpsPort}
			log.Fatal(sc.StartTLS(context.Background(), s.e, certPEM, keyPEM))
		}()
	}

	fmt.Printf(banner, "=> http :"+port)
	log.Fatal(s.e.Start(":" + port))
}
