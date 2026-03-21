package main

import (
	"SMOE/moe"
)

func main() {
	s := moe.New()
	s.LoadMiddlewareRoutes()
	s.Listen()
}
