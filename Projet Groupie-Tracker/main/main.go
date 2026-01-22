package main

import (
	"fmt"
	"log"
	"net/http"

	"groupie_tracker/router"
)

// main démarre le serveur HTTP de l'application.
// Il crée le routeur, affiche l'URL d'écoute et lance `http.ListenAndServe`.
func main() {
	mux := router.New()
	addr := ":8080"
	fullURL := "http://localhost" + addr

	
	fmt.Println(fullURL)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
