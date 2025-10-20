#!/bin/bash
# Generate optimized Open Graph image for social media (1200x627)

INPUT_IMAGE="static/kind-and-curious-ZDUXvlyU_iI-unsplash.jpg"
OUTPUT_IMAGE="static/kind-and-curious-ZDUXvlyU_iI-unsplash-og.jpg"

# LinkedIn/Facebook recommended size: 1200x627 (aspect ratio ~1.91:1)
# This will:
# 1. Resize to fill 1200x627 (covering the entire area)
# 2. Center crop to exactly 1200x627
# 3. Optimize quality for web

magick "$INPUT_IMAGE" \
  -resize 1200x627^ \
  -gravity center \
  -extent 1200x627 \
  -quality 85 \
  "$OUTPUT_IMAGE"

echo "✓ Generated Open Graph image: $OUTPUT_IMAGE"
magick identify "$OUTPUT_IMAGE"
