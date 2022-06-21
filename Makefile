addDocument:
	curl -X POST -H 'Content-Type: application/json' -d '{"name": "albert", "age": "30"}' http://localhost:8080/docs

getID:
	curl http://localhost:8080/docs/$(ID)

getDocuments:
	curl http://localhost:8080/docs

query:
	curl --get http://localhost:8080/docs --data-urlencode 'q=$(QUERY)'