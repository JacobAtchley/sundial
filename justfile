set shell := ["bash", "-euo", "pipefail", "-c"]

export CGO_CFLAGS := "-O2 -g -mmacosx-version-min=14.0"
export CGO_LDFLAGS := "-mmacosx-version-min=14.0"

plist := justfile_directory() / "Info.plist"
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
ldflags := "-X main.version=" + version + " -linkmode=external -extldflags '-sectcreate __TEXT __info_plist " + plist + "'"
bindir := env_var_or_default("SUNDIAL_BIN_DIR", home_directory() / ".local" / "bin")

default:
    @just --list

# Build bin/sundial with the embedded Info.plist
build:
    go build -ldflags "{{ldflags}}" -o bin/sundial ./cmd/sundial

# Build and run with arguments
run *args: build
    ./bin/sundial {{args}}

# Run with fictional demo data (safe for screenshots)
demo *args: build
    SUNDIAL_SOURCE=fake ./bin/sundial {{args}}

# Unit and golden tests (no calendar access needed)
test:
    go test ./...

# Update golden files after an intentional UI change
golden:
    go test ./internal/cli/ ./internal/tui/agenda/ ./internal/tui/month/ ./internal/tui/week/ ./internal/tui/detail/ -update

# EventKit integration tests (needs calendar access; logs counts only)
test-integration:
    go test -tags integration -ldflags "{{ldflags}}" -count=1 -v ./internal/eventkit/...

lint:
    go tool golangci-lint run

fmt:
    gofmt -w .
    go tool goimports -w -local github.com/JacobAtchley/sundial .

# Install bin/sundial into ~/.local/bin (override with SUNDIAL_BIN_DIR)
install: build
    mkdir -p "{{bindir}}"
    install -m 0755 bin/sundial "{{bindir}}/sundial"
    @echo "Installed {{bindir}}/sundial"
    @case ":$PATH:" in *":{{bindir}}:"*) ;; *) echo "Note: {{bindir}} is not on your PATH";; esac

# Remove the installed binary
uninstall:
    rm -f "{{bindir}}/sundial"

# Record docs/demo.gif with fictional data (requires vhs)
record: build
    vhs demo.tape

clean:
    rm -rf bin dist
