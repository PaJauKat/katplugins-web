package main

import (
	"log"
	"net/http"
	"os"

	"github.com/PaJauKat/katplugins-web/api/internal/config"
	"github.com/PaJauKat/katplugins-web/api/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.New(cfg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("katplugins api escuchando en %s", addr)

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
