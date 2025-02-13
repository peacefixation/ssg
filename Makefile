run:
	go run cmd/ssg/main.go -watch -serve

init:
	mkdir -p config content/posts static/css
	cat "title: New Site\nsyntax-highlight-style: monokai\n" > config/site.yaml

new-post:
	./scripts/new-post.sh

links:
	go run cmd/ssg/main.go -links

clean:
	rm -r output
