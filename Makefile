BINARY := digitron

# -s -w leaves out the symbol table and debug information, which roughly
# halves the size of the binary.
LDFLAGS := -s -w

.PHONY: build run test vet screenshot clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

run: build
	./$(BINARY)

# Every run of the tests has a timeout, so that one that hangs fails with
# the stacks of what it was waiting on, rather than waiting for ever.
TEST_TIMEOUT := 5m

test:
	go test -race -timeout $(TEST_TIMEOUT) ./...

vet:
	go vet ./...

# Renders sample calculator states (3x scale) as PNGs into docs/previews.
screenshot:
	mkdir -p docs/previews
	PREVIEW_DIR=$(CURDIR)/docs/previews go test -count=1 -timeout $(TEST_TIMEOUT) -run '^TestPreview$$' ./ui/

clean:
	rm -f $(BINARY)
