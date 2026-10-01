set shell := ["bash", "-euo", "pipefail", "-c"]

export CGO_CFLAGS := "-O2 -g -mmacosx-version-min=14.0"
export CGO_LDFLAGS := "-mmacosx-version-min=14.0"

plist := justfile_directory() / "Info.plist"
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
ldflags := "-X main.version=" + version + " -linkmode=external -extldflags '-sectcreate __TEXT __info_plist " + plist + "'"

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

# Install into $GOBIN with the embedded Info.plist
install:
    go install -ldflags "{{ldflags}}" ./cmd/sundial

clean:
    rm -rf bin dist
