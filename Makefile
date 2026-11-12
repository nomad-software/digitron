BINARY := digitron

# -s -w leaves out the symbol table and debug information, which roughly
# halves the size of the binary.
LDFLAGS := -s -w

.PHONY: build run test bench fuzz vet screenshot appimage clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

run: build
	./$(BINARY)

# Every run of the tests has a timeout, so that one that hangs fails with
# the stacks of what it was waiting on, rather than waiting for ever.
TEST_TIMEOUT := 5m

# Playing sound through Qt Multimedia crashes the race detector as the test
# program exits, so the sound's playing test runs without -race, by itself.
test:
	go test -race -timeout $(TEST_TIMEOUT) ./...
	go test -timeout $(TEST_TIMEOUT) -run '^TestPlay$$' ./sound

# Times the Qt layer (painting, a key, a click), to compare Qt bindings.
bench:
	go test -run '^$$' -bench . -benchmem ./gui

# Presses keys at random, looking for a state the engine must never be in.
# go test runs only FuzzKeys's seeds; this searches for FUZZTIME.
FUZZTIME ?= 60s

fuzz:
	go test -run '^$$' -fuzz '^FuzzKeys$$' -fuzztime $(FUZZTIME) ./calc

vet:
	go vet ./...

# Renders sample calculator states (3x scale) as PNGs into docs/previews.
screenshot:
	mkdir -p docs/previews
	PREVIEW_DIR=$(CURDIR)/docs/previews go test -count=1 -timeout $(TEST_TIMEOUT) -run '^TestPreview$$' ./ui/

# Packs the program and the Qt it uses into one file, built in APPIMAGE_BUILD
# and left in the project's folder. Releases are made by the GitHub workflow,
# on an old distribution so that they run on more systems.
APPIMAGE_BUILD := /tmp/digitron-build

appimage:
	BUILD=$(APPIMAGE_BUILD) packaging/appimage.sh

clean:
	rm -f $(BINARY) Digitron-*.AppImage
	rm -rf $(APPIMAGE_BUILD)
