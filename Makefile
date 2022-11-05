install-swag:
	go get -u github.com/swaggo/swag/cmd/swag

start-docs:
	docker run -d -p 80:8080 \
		--name=swagger-ui \
		-e SWAGGER_JSON=/www/swagger.json \
		-v ${PWD}/docs:/www \
		swaggerapi/swagger-ui

stop-docs:
	docker stop swagger-ui

generate-swagger-docs:
	swag init -g ./main.go --output ./docs