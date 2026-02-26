.PHONY: build run test clean docker-build docker-run lint vet fmt

BINARY=picolm-server
CONFIG=config.yaml
DOCKER_IMAGE=picolm-server
DOCKER_TAG=latest

build:
	go build -o $(BINARY) ./cmd/server/

run: build
	./$(BINARY) -config $(CONFIG)

test:
	go test -v ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

lint: vet fmt

clean:
	rm -f $(BINARY)

docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run:
	docker run --rm -p 8080:8080 -v $(CONFIG):/app/config.yaml $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose:
	docker-compose up -d

docker-compose-down:
	docker-compose down
