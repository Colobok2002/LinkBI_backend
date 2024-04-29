.PHONY: run
run: swagger
	@go run main.go

.PHONY: swagger
swagger:
	@swag init

.PHONY: buildM
buildM:
	@go build -o build/main_build     

.PHONY: buildD
buildD:
	@go build -o build/dev_build      

.PHONY: prod
prod:
	@./build/main_build  

