.PHONY: build serve watch test add-item add-list

build:
	go build .

serve:
	go run . serve

watch:
	go run . serve --watch

test:
	go test ./...

new-item:
	go run . new item

new-list:
	go run . new list
