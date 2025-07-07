#!/bin/bash

# Configuration
PLAYER="spotify" # Hardcode your preferred player here
CACHE_DIR="${HOME}/.cache/mpris-cover"
mkdir -p "${CACHE_DIR}"

# Process metadata changes
playerctl metadata -F --player "${PLAYER}" --format '{{ mpris:artUrl }}' | while read -r url; do
    [ -z "${url}" ] && continue

    # Generate cache filename (MD5 -> base64 URL-safe)
    hash=$(echo -n "${url}" | md5sum | awk '{print $1}' | xxd -r -p | base64 | tr -d '=' | tr '+/' '-_')
    cached_file="${CACHE_DIR}/${hash}.png"

    # Return cached file if exists
    if [ -f "${cached_file}" ]; then
        echo "${cached_file}"
        continue
    fi

    # Create temp file
    tmp_dl=$(mktemp)
    trap 'rm -f "${tmp_dl}"' EXIT

    # Download cover art
    if ! curl -sSL "${url}" -o "${tmp_dl}"; then
        echo "ERROR: Download failed: ${url}" >&2
        continue
    fi

    # Verify image type
    mime_type=$(file -b --mime-type "${tmp_dl}")
    if [[ ! "${mime_type}" =~ ^image/ ]]; then
        echo "ERROR: Not an image: ${mime_type}" >&2
        continue
    fi

    # Process image with rounding effect and resize
    if ! convert "${tmp_dl}" \
        \( +clone -alpha extract \
        -draw "fill black polygon 0,0 0,30 30,0 fill white circle 30,30 30,0" \
        \( +clone -flip \) -compose Multiply -composite \
        \( +clone -flop \) -compose Multiply -composite \) \
        -alpha off -compose CopyOpacity -composite \
        -resize 200x \
        "${cached_file}"; then
        echo "ERROR: Image processing failed" >&2
        continue
    fi

    echo "${cached_file}"
done
