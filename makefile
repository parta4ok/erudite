lint:
	golangci-lint run -c ./.golangci.yml ./...
	@echo "\033[1;34m"lintering finished
test:
	go test -failfast -race ./...
	@echo "\033[0;32mall unit tests passed"
stamp:
	date +%s
local:
	docker-compose --profile services up --build -d
local_down:
	docker-compose down --volumes --remove-orphans
compose_up:
	docker-compose --profile services up --build -d
compose_down:
	docker-compose down --volumes --remove-orphans