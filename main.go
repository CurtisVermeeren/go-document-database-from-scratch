package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Create a new server instance
	s, err := NewServer("docdb.data", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer s.db.Close()

	// Index the database
	log.Println("Indexing DB...")
	s.reindex()
	log.Println("Done Indexing.")

	// Create the router and setup routes
	router := mux.NewRouter()
	router.HandleFunc("/docs", s.AddDocument).Methods("POST")
	router.HandleFunc("/docs", s.SearchDocuments).Methods("GET")
	router.HandleFunc("/docs/{id}", s.GetDocument).Methods("GET")

	// Run the server
	log.Println("Listening on ", s.port)
	log.Fatal(http.ListenAndServe(s.port, router))
}
