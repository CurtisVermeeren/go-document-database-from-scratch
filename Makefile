addDocument:
	curl -X POST -H 'Content-Type: application/json' -d '{"name": "Kevin", "age": "45"}' http://localhost:8080/docs

getID:
	curl http://localhost:8080/docs/$(ID)

getDocuments:
	curl http://localhost:8080/docs

queryName:
	curl --get http://localhost:8080/docs --data-urlencode 'q=name:$(NAME)'

queryAge:
	curl --get http://localhost:8080/docs --data-urlencode 'q=age:$(AGE)'

