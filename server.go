package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	db      *pebble.DB // Primary DB
	indexDb *pebble.DB // Indexing DB
	port    string
}

func NewServer(database string, port string) (*Server, error) {
	s := Server{
		db:   nil,
		port: port,
	}
	var err error
	// Open primary db
	s.db, err = pebble.Open(database, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	// Open indexing db
	s.indexDb, err = pebble.Open(database+".index", &pebble.Options{})
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

	// Index the new document in indexDb
	s.index(id, document)

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

// SearchDocuments takes a search query from the url and returns a JSON response of all documents matching
// SearchDocuments makes use of the index database
func (s Server) SearchDocuments(w http.ResponseWriter, r *http.Request) {
	// Parse the query from the url
	q, err := parseQuery(r.URL.Query().Get("q"))
	if err != nil {
		jsonResponse(w, nil, err)
		return
	}

	// isRange tracks if the argument is equality or a range value using < and >
	isRange := false

	// idsArgumentCount is a map of ids to count. It tracks the number of times an id has been found
	idsArgumentCount := map[string]int{}

	// nonRangeArguments tracks the number of eqality arguments found
	nonRangeArguments := 0

	for _, argument := range q.ands {
		// If the argument is an eqaulity check
		if argument.op == "=" {
			nonRangeArguments++
			// Get all ids that correspond to the key value pair
			ids, err := s.lookup(fmt.Sprintf("%s=%v", strings.Join(argument.key, "."), argument.value))
			if err != nil {
				jsonResponse(w, nil, err)
				return
			}

			// iterate over all ids found for the key value pair
			for _, id := range ids {
				// If the id is not found set its count to 0
				_, ok := idsArgumentCount[id]
				if !ok {
					idsArgumentCount[id] = 0
				}
				// If the id was found increate the id count by 1
				idsArgumentCount[id]++
			}
		} else {
			isRange = true
		}
	}

	// idsInAll is a slice of ids that were found to match ALL eqaulity arguments
	var idsInAll []string
	for id, count := range idsArgumentCount {
		// The id matching count equals the total number of equality arguments checked
		if count == nonRangeArguments {
			idsInAll = append(idsInAll, id)
		}
	}

	// documents contains all matching documents
	var documents []any

	// If skipIndex was set in the query remove the ids found using indexing
	if r.URL.Query().Get("skipIndex") == "true" {
		idsInAll = nil
	}

	// If indexing was used
	if len(idsInAll) > 0 {
		// For all matching ids so far get the document from the primary database
		for _, id := range idsInAll {
			document, err := s.GetDocumentById([]byte(id))
			if err != nil {
				jsonResponse(w, nil, err)
				return
			}

			// If there is no range part to the query add the matching document
			// If there is a range part to the query check if the document matches the query then add it
			if !isRange || q.match(document) {
				documents = append(documents, map[string]any{
					"id":   id,
					"body": document,
				})
			}
		}
	} else {
		// No indexing was used so iterate over the primary database to check for matching documents
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

	}
	// Return all documents as a JSON response
	jsonResponse(w, map[string]any{"documents": documents, "count": len(documents)}, nil)
}

// lookup a path value pair (a.b.c=val) in the indexDb
// returns a slice of ids that belong to the path value pair
func (s Server) lookup(pathValue string) ([]string, error) {
	idsString, closer, err := s.indexDb.Get([]byte(pathValue))
	if err != nil && err != pebble.ErrNotFound {
		return nil, fmt.Errorf("could not look up pathvalue [%#v]: %s", pathValue, err)
	}

	if closer != nil {
		defer closer.Close()
	}

	if len(idsString) == 0 {
		return nil, nil
	}

	return strings.Split(string(idsString), ","), nil
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
