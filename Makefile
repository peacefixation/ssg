.PHONY: build serve watch test

build:
	go build .

serve:
	go run . serve

watch:
	go run . serve --watch

test:
	go test ./...

add:
	go run . add