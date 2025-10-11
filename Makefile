.PHONY: run build clean highlight

.DEFAULT_GOAL := run

run: build
	cd example && ../aether  && cd ..

build:
	go build .

highlight:
	@echo "Building syntax highlighter..."
	@go build -o highlight-cdc ./cmd/highlight
	@echo "\nHighlighting example/cadence/transactions/swap.cdc...\n"
	@./highlight-cdc example/cadence/transactions/swap.cdc

clean:
	rm -f aether highlight-cdc
