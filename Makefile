BINARY := digitron

# -s -w leaves out the symbol table and debug information, which roughly
# halves the size of the binary.
LDFLAGS := -s -w

.PHONY: build run test vet screenshot icons appimage macos clean

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

# Draws the macOS app's icon, packaging/macos/digitron.png, from the logo,
# within Apple's margin. A test says when the logo has changed since it was
# run.
icons:
	DIGITRON_ICONS=$(CURDIR)/packaging go test -count=1 -run '^TestWriteIcons$$' ./ui/

# Packs the program and the Qt it uses into one file. Releases are made by the
# GitHub workflow, on an old distribution so that they run on more systems.
appimage:
	packaging/appimage.sh

# Builds the macOS app in a disk image, on a Mac with Homebrew's Qt.
macos:
	packaging/macos.sh

clean:
	rm -f $(BINARY)
	rm -rf build
