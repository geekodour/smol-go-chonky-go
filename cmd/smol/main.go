package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	// CRUD endpoints for cats
	// mux.HandleFunc("/cats", getAllCats)
	// mux.HandleFunc("/cats/add", addCat)
	// mux.HandleFunc("/cats/update", updateCat)
	// mux.HandleFunc("/cats/delete", deleteCat)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on :%s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
