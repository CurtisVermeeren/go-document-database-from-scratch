package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

/*
A simplification of Lucene. There will only be key-value matches.
Field names and field values can be quoted.
They must be quoted if they contain spaces or colons, among other things.
Key-value matches are separated by whitespace.
The can onlyt be AND-ed together and that is done implicitly.

An example of some valid filters:
a:1
b:fifteen a:<3
a.b:12
title:"Which way?"
" a key 2":tenant
" flubber ":"blubber "

Nested paths are specified using JSON path syntax
EX a.b would retrieve 4 from the following
{"a": {"b": 4, "d": 100}, "c": 8})
*/

// queryComparions represents a single part of a query
// A key represents a set of keys in json syntax to reach a value
// op is the operation used to compare the query value with database values
type queryComparison struct {
	key   []string
	value string
	op    string
}

// query is a collection of queryComparisons to be performed
type query struct {
	ands []queryComparison
}

// parseQuery takes a string and builds a query
// a query contains one or more queryComparison
// each queryComparison is a key and value pairing that is parsed. A questComparison can have 3 options : meaning equal, :< meaning less than, :> meaning greater than
func parseQuery(q string) (*query, error) {
	// If no query is found return nil
	if q == "" {
		return &query{}, nil
	}

	i := 0
	var parsed query
	var qRune = []rune(q)
	// Parse key, value pairings in the string into an overall list of AND-ed arguments to make up a query
	for i < len(qRune) {
		// Ignore whitespace
		for unicode.IsSpace(qRune[i]) {
			i++
		}

		// Get the key
		key, nextIndex, err := lexString(qRune, i)
		if err != nil {
			return nil, fmt.Errorf("expected valid key, got [%s]: `%s`", err, q[nextIndex:])
		}

		if q[nextIndex] != ':' {
			return nil, fmt.Errorf("expected colon at %d, got `%x`", nextIndex, q[nextIndex])
		}
		i = nextIndex + 1

		// Get the operations if there is one
		// can be :, :> or :<
		op := "="
		if q[i] == '>' || q[i] == '<' {
			op = string(q[i])
			i++
		}

		// Get the value
		value, nextIndex, err := lexString(qRune, i)
		if err != nil {
			return nil, fmt.Errorf("expected valid value, go [%s]: `%x`", err, q[nextIndex])
		}
		i = nextIndex

		argument := queryComparison{key: strings.Split(key, "."), value: value, op: op}
		parsed.ands = append(parsed.ands, argument)
	}

	return &parsed, nil
}

// lexString takes a slice of runes to be lexed and an index of the starting point
// returns a string of the text between two quotes, or the sequence of contiguous letters, digits, and . starting from the index
// returns an integer representing the end index of the sequence
func lexString(input []rune, index int) (string, int, error) {
	// Check that index is inside the input
	if index >= len(input) {
		return "", index, nil
	}

	// If the sequence starts with a quote
	// TODO: Handle nested quotes
	if input[index] == '"' {
		index++
		foundEnd := false

		var s []rune
		// append all characters between quotes
		for index < len(input) {
			if input[index] == '"' {
				foundEnd = true
				break
			}

			s = append(s, input[index])
			index++
		}

		if !foundEnd {
			return "", index, fmt.Errorf("expected end of quoted string")
		}

		// return the string of characters between quotes, and the index after the end quote
		return string(s), index + 1, nil
	}

	// If the starting index is unquoted then we read the sequence as contiguous digits/letters
	var s []rune
	var c rune

	for index < len(input) {
		c = input[index]
		// If the character is not a letter, digit or period then break
		if !(unicode.IsLetter(c) || unicode.IsDigit(c) || c == '.') {
			break
		}
		// Append the character to the sequence
		s = append(s, c)
		index++
	}

	// Check if a string was found
	if len(s) == 0 {
		return "", index, fmt.Errorf("no string found")
	}

	return string(s), index, nil
}

func (q query) match(doc map[string]any) bool {
	// Check each argument in the query
	for _, argument := range q.ands {
		// Attempt to get the value from the document that maps to the path of keys in the query
		value, ok := getPath(doc, argument.key)
		if !ok {
			return false
		}

		// Check if value is equal to the query value if the operation is =
		// If the values do not match then the query fails
		if argument.op == "=" {
			match := fmt.Sprintf("%v", value) == argument.value
			if !match {
				return false
			}
			continue
		}

		// Get the right sided argument as float64
		right, err := strconv.ParseFloat(argument.value, 64)
		if err != nil {
			return false
		}

		// Get the left sided argument as float64
		// switch is needed in Go to ensure for all possible starting types
		var left float64
		switch t := value.(type) {
		case float64:
			left = t
		case float32:
			left = float64(t)
		case uint:
			left = float64(t)
		case uint8:
			left = float64(t)
		case uint16:
			left = float64(t)
		case uint32:
			left = float64(t)
		case uint64:
			left = float64(t)
		case int:
			left = float64(t)
		case int8:
			left = float64(t)
		case int16:
			left = float64(t)
		case int32:
			left = float64(t)
		case int64:
			left = float64(t)
		case string:
			left, err = strconv.ParseFloat(t, 64)
			if err != nil {
				return false
			}
		default:
			return false
		}

		// Check which value is greater based on the query parameter of > or <
		// If the value is not larger or smaller as intended then the query fails
		if argument.op == ">" {
			if left <= right {
				return false
			}
			continue
		}

		if left >= right {
			return false
		}
	}

	return true
}

// getPath takes a document doc, and a queryComparion key of type []string
// the key consists of the key names in a path
func getPath(doc map[string]any, parts []string) (any, bool) {
	// docSegment holds the map[string]any as we traverse down the path
	var docSegment any = doc
	// Check each part of the key path from start to end
	for _, part := range parts {
		// Get the new segment of the map
		m, ok := docSegment.(map[string]any)
		if !ok {
			return nil, false
		}
		// Check if the current path key continues
		if docSegment, ok = m[part]; !ok {
			return nil, false
		}
	}

	// Return the final value and true if the path of keys was successfully traversed
	return docSegment, true
}
