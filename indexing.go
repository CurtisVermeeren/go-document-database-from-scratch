package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cockroachdb/pebble"
)

// index is used to store path and value pairings in the indexDB
// takes a document of form map[string]any and the id string of that document
// indexDb uses the json syntax path values, example : a.b.d=2, as the keys
// and a list of comma separated document ids as the value
func (s Server) index(id string, document map[string]any) {
	pv := getPathValues(document, "")

	for _, pathValue := range pv {
		// Try to get the path from the indexDb
		idsString, closer, err := s.indexDb.Get([]byte(pathValue))
		if err != nil && err != pebble.ErrNotFound {
			log.Printf("Could not look up pathvalue [%#v]: %s", document, err)
			return
		}

		// If nothing was found in indexDb matching the path
		if len(idsString) == 0 {
			// Add the documents id
			idsString = []byte(id)
		} else {
			ids := strings.Split(string(idsString), ",")

			// Check for the documents id in the found list of ids for the key
			found := false
			for _, existingId := range ids {
				if id == existingId {
					found = true
				}
			}

			// If the id was not found in the set of ids then add it
			if !found {
				idsString = append(idsString, []byte(","+id)...)
			}
		}

		// Attempt to close the returned slice from indexDb.Get
		if closer != nil {
			err = closer.Close()
			if err != nil {
				log.Printf("could not close: %s", err)
				return
			}
		}

		// Set the value in indexDb
		err = s.indexDb.Set([]byte(pathValue), idsString, pebble.Sync)
		if err != nil {
			log.Printf("could not update index: %s", err)
			return
		}
	}
}

// getPathValues takes a map[string]any that represents a document as obj
// and a prefix string that recursively tracks the current path
// returns a slive of strings of all paths found for a documents structure
// paths are of the json format EX alpha.beta.numbers.a=1 would be represented in the following
// {"alpha": {"beta": {"numbers":{a:1,b:3,c:4}}}}
func getPathValues(obj map[string]any, prefix string) []string {
	var pvs []string

	// Iterate over the map
	for key, val := range obj {
		// If a value isn't an array
		switch t := val.(type) {
		// Recursively get the path for that value
		// Use the key as the prefix for the next path
		case map[string]any:
			pvs = append(pvs, getPathValues(t, key)...)
			continue
		case []interface{}:
			// Skip arrays
			continue
		}

		// Construct the key using the values of the prefix
		if prefix != "" {
			key = prefix + "." + key
		}

		// Create a string of the form path=value and add it to the pathvalue returned for the document
		pvs = append(pvs, fmt.Sprintf("%s=%v", key, val))

	}
	return pvs
}

// reindex is used to index all existing documents
func (s Server) reindex() {
	iter := s.db.NewIter(nil)
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		var document map[string]any
		err := json.Unmarshal(iter.Value(), &document)
		if err != nil {
			log.Printf("Unable to parse bad document, %s: %s", string(iter.Key()), err)
			continue
		}
		s.index(string(iter.Key()), document)
	}
}
