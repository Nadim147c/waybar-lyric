#!/usr/bin/env bash

set -e # Exit on any error

# Configuration
APP_NAME="waybar-lyric"
BUILD_DIR="build"
APP_BIN="${BUILD_DIR}/${APP_NAME}"
VERSION=$(git describe --tags --abbrev=0)
COMPLETION_DIR="${BUILD_DIR}/completions"

go install github.com/charmbracelet/gum@latest

gum format -- "# Building $APP_NAME version $VERSION"

# Step 1: Compile for native arch and OS
gum format -- "1. Building native binary..."
gum spin --spinner dot --title "Building native binary" -- \
    go build -o "$APP_BIN" .

# Step 2: Generate completions
gum format -- "2. Generating shell completions..."
mkdir -p "$COMPLETION_DIR"
$APP_BIN _carapace bash >"$COMPLETION_DIR/$APP_NAME.bash"
$APP_BIN _carapace zsh >"$COMPLETION_DIR/$APP_NAME.zsh"
$APP_BIN _carapace fish >"$COMPLETION_DIR/$APP_NAME.fish"

# Step 3: Cross-compile for different platforms
gum format -- "3. Cross-compiling for different platforms..."

echo ""

# Define target platforms (OS/ARCH pairs)
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r os arch <<<"$platform"

    # Set output binary name
    binary_name="$APP_NAME"
    if [ "$os" = "windows" ]; then
        binary_name="$APP_NAME.exe"
    fi

    # Create directory structure
    package_dir="$BUILD_DIR/${APP_NAME}-${VERSION}-${os}-${arch}"
    bin_dir="$package_dir/usr/bin"
    share_base_dir="$package_dir/usr/share"

    # Cross-compile
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
        gum spin --spinner dot --title "Building for $os/$arch..." -- \
        make build SRC_BIN="$bin_dir/$binary_name" VERSION="$VERSION"

    # Copy completion files
    install -Dm644 "$COMPLETION_DIR/$APP_NAME.bash" "$share_base_dir/bash-completion/completions/$APP_NAME"
    install -Dm644 "$COMPLETION_DIR/$APP_NAME.zsh" "$share_base_dir/zsh/site-functions/_$APP_NAME"
    install -Dm644 "$COMPLETION_DIR/$APP_NAME.fish" "$share_base_dir/fish/vendor_completions.d/$APP_NAME.fish"

    cd "$BUILD_DIR"
    gum spin --spinner dot --title "Creating archive for $os/$arch..." -- \
        tar -czf "${APP_NAME}-${VERSION}-${os}-${arch}.tar.gz" "$(basename "$package_dir")"
    cd - >/dev/null

    gum log -sl info --prefix "$os/$arch" "Build complete"
done

# Step 5: Generate SHA256 checksums
gum format -- "5. Generating SHA256 checksums..."
echo ""
cd "$BUILD_DIR"
for archive in *.tar.gz; do
    if [ -f "$archive" ]; then
        sha256sum "$archive" >"${archive}.sha256"
        gum log -sl info "Created checksum for $archive"
    fi
done
cd - >/dev/null

gum format "# All artifacts are in the $BUILD_DIR directory:"
printf ' - %s\n' "$BUILD_DIR"/*.tar.gz* | gum format

gum format "# Build completed successfully!"
