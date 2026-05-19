.PHONY: compile site serve watch test new-item new-list

compile:
	go build .

site:
	go run . build

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
