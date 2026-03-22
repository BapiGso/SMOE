package moe

import (
	"fmt"
	"log"
)

func (s *Smoe) Listen() {
	port := s.cfg.Port
	if port == "" {
		port = "95"
	}
	fmt.Printf(banner, "=> http :"+port)
	log.Fatal(s.e.Start(":" + port))
}
