# Document Database in Go from Scratch
Files for a VSCode devcontainer using Go. 
Includes the Go Extension and the tools need for it in the Dockerfile.
The default terminal is Zsh as configured in the Dockerfile

https://notes.eatonphil.com/documentdb.html

## Seeding Database
seed.sh is a script used to seed the database with the movies.json file

Run the command `chmod +x seed.sh` then `./seed.sh movies.json`

## Makefile
The makefile contains helper targets for making curl commands

## Querying the documents
A simplification of Lucene. There will only be key-value matches.
Field names and field values can be quoted.
They must be quoted if they contain spaces or colons, among other things.
Key-value matches are separated by whitespace.
The can onlyt be AND-ed together and that is done implicitly.

An example of some valid filters:

- a:1
- b:fifteen a:<3
- a.b:12
- title:"Which way?"
- " a key 2":tenant
- " flubber ":"blubber "

Nested paths are specified using JSON path syntax
EX a.b would retrieve 4 from the following
{"a": {"b": 4, "d": 100}, "c": 8})

Using the movies.json database `curl -s --get http://localhost:8080/docs --data-urlencode 'q="title":"Batman"'` would return something like:

```{"body":{"count":1,"documents":[{"body":{"cast":["Lewis Wilson","Douglas Croft","J. Carrol Naish","Shirley Patterson"],"genres":["Action"],"title":"Batman","year":1943},"id":"8e6a52b6-950e-461f-ba6f-0902aec93572"}]},"status":"ok"}```

## Indexing
For every path in a document (That isn't a path within an array) store the path and the value of the document at that path.