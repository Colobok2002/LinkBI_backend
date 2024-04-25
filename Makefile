.PHONY: run
run: swagger
	@go run main.go

.PHONY: swagger
swagger:
	@swag init
