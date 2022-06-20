package main

import (
	"encoding/json"
	"net/http"

	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	db   *pebble.DB
	port string
}

func NewServer(database string, port string) (*Server, error) {
	s := Server{
		db:   nil,
		port: port,
	}
	var err error
	s.db, err = pebble.Open(database, &pebble.Options{})
	return &s, err
}

// AddDocument parses a post request body and creates a new document from it
// shows the uuid of the new document
func (s Server) AddDocument(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var document map[string]any
	err := dec.Decode(&document)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	// New unique id for the document
	id := uuid.New().String()

	bs, err := json.Marshal(document)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}
	err = s.db.Set([]byte(id), bs, pebble.Sync)
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	jsonResponse(w, map[string]any{
		"id": id,
	}, nil)
}

func (s Server) SearchDocuments(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r.URL.Query().Get("q"))
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	var documents []map[string]any

	iter := s.db.NewIter(nil)
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		var document map[string]any
		err = json.Unmarshal(iter.Value(), &document)
		if err != nil {
			jsonResponse(w, nil, err)
			return
		}

		if q.match(document) {
			documents = append(documents, map[string]any{
				"id":   string(iter.Key()),
				"body": document,
			})
		}
	}
	jsonResponse(w, map[string]any{"documents": documents, "count": len(documents)}, nil)
}

// GetDocument gets a uuid from the url and returns the document as JSON
func (s Server) GetDocument(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	document, err := s.GetDocumentById([]byte(id))
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	jsonResponse(w, map[string]any{
		"document": document,
	}, nil)
}

// GetDocumentById takes an id and returns a map of the document it matches
func (s Server) GetDocumentById(id []byte) (map[string]any, error) {
	valBytes, closer, err := s.db.Get(id)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var document map[string]any
	err = json.Unmarshal(valBytes, &document)
	return document, err
}
