
#!/bin/bash
set -e

# Dependencies:
#   pandoc       - apt install pandoc
#   xelatex      - apt install texlive-xetex
#   DejaVu fonts - apt install fonts-dejavu

PWDIR="$PWD"

SCRIPT_DIR="$(dirname "$(realpath "$0")")"

cd "$SCRIPT_DIR"

pandoc "$SCRIPT_DIR/API-docs.md" -o "$SCRIPT_DIR/API-docs.pdf" \
  --pdf-engine=xelatex \
  --toc \
  --toc-depth=3 \
  -V geometry:margin=2.5cm \
  -V fontsize=11pt \
  -V mainfont="DejaVu Serif" \
  -V monofont="DejaVu Sans Mono" \
  -V linestretch=1.4 \
  -V colorlinks=true \
  -V toccolor=blue \
  -V linkcolor=blue

cd "$PWDIR"

echo "Generated: $SCRIPT_DIR/API-docs.pdf"
