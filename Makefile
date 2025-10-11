.PHONY: run build clean highlight

.DEFAULT_GOAL := run

run: build
	cd example && ../aether  && cd ..

build:
	go build .

clean:
	rm -f aether 
