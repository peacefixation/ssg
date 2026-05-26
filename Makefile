.PHONY: compile site serve watch test new-item new-list clean resize-photos

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
	go run . new item $(ARGS)

new-list:
	go run . new list $(ARGS)

clean:
	rm -r public
	rm -r cache/*

resize-photos:
	for f in $(DIR)/*.JPEG; do convert "$$f" -resize 50% "$$f"; done
