NAME = waybar-lyric

GO      ?= go
REVIVE  ?= revive
SRC_BIN ?= bin/$(NAME)
PREFIX  ?= /usr/local
VERSION ?= $(shell git describe --tags)

BIN_DIR         = $(shell realpath -m "$(PREFIX)/bin")
BIN_FILE        = $(shell realpath -m "$(BIN_DIR)/$(NAME)")
DOC_DIR         = $(shell realpath -m "$(PREFIX)/share/doc/$(NAME)")
DOC_FILE        = $(shell realpath -m "$(PREFIX)/share/doc/$(NAME)/README.md")
LICENSE_DIR     = $(shell realpath -m "$(PREFIX)/share/licenses/$(NAME)")
LICENSE_FILE    = $(shell realpath -m "$(PREFIX)/share/licenses/$(NAME)/LICENSE")
BASH_COMPLETION = $(shell realpath -m "$(PREFIX)/share/bash-completion/completions/$(NAME)")
ZSH_COMPLETION  = $(shell realpath -m "$(PREFIX)/share/zsh/site-functions/_$(NAME)")
FISH_COMPLETION = $(shell realpath -m "$(PREFIX)/share/fish/vendor_completions.d/$(NAME).fish")

-include Makefile.local

# Default target
.PHONY: all
all: build

# Build the Go binary
.PHONY: build
build:
	$(GO) build -trimpath -ldflags '-X main.Version=$(VERSION)' -o $(SRC_BIN)

# Build the Go binary
.PHONY: test
test:
	$(GO) test -v -cover ./...
	$(REVIVE) -config revive.toml

# Clean up build artifacts
.PHONY: clean
clean:
	rm -f $(SRC_BIN)

.PHONY: install
install: check-path
	install -Dsm755 $(SRC_BIN) "$(BIN_FILE)"
	install -Dm644  LICENSE    "$(LICENSE_FILE)"
	install -Dm644  README.md  "$(DOC_FILE)"

	$(SRC_BIN) _carapace bash | install -Dm644 /dev/stdin "$(BASH_COMPLETION)"
	$(SRC_BIN) _carapace zsh  | install -Dm644 /dev/stdin "$(ZSH_COMPLETION)"
	$(SRC_BIN) _carapace fish | install -Dm644 /dev/stdin "$(FISH_COMPLETION)"

.PHONY: check-path
check-path:
	@if ! paths="$$(echo $$PATH | xargs -d: -n1 realpath -m 2>/dev/null)"; then \
		echo "❌ Failed to resolve PATH entries."; \
	elif ! echo "$$paths" | grep -qxF "$(BIN_DIR)"; then \
		echo "⚠️  Warning: $(BIN_DIR) is not in your PATH."; \
		echo "   Add it with:"; \
		echo "     export PATH=\"$(BIN_DIR):\$$PATH\""; \
	fi

.PHONY: uninstall
uninstall:
	@rm -vrf $(BIN_FILE) $(LICENSE_DIR) $(DOC_DIR) $(BASH_COMPLETION) $(ZSH_COMPLETION) $(FISH_COMPLETION)
