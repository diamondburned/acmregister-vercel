package main

import (
	"log"
	"net/http"
	"os"

	handler "github.com/diamondburned/acmregister-vercel/api"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/interaction", handler.HandleInteraction)

	log.Println("serving at", os.Args[1])
	log.Fatalln(http.ListenAndServe(os.Args[1], mux))
}
