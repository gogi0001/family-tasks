package main

import (
	"log"
	"net/http"
	"os"

	"github.com/<ваш-юзер>/family-tasks/internal/api"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	router := api.NewRouter()

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}