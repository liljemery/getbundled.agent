#!/usr/bin/env bash
# Manual agent release when CI is not triggered (tag push / workflow_dispatch).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-$(grep 'const AgentVersion' "$ROOT/internal/config/config.go" | sed -E 's/.*"([^"]+)".*/\1/')}"
ARTIFACT="getbundled-agent-${VERSION}-linux-amd64"
OUT="${TMPDIR:-/tmp}/${ARTIFACT}"

echo "Building ${VERSION} → ${OUT}"
(
  cd "$ROOT"
  GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$OUT" ./cmd/getbundled-agent
)
SHA="$(sha256sum "$OUT" | awk '{print $1}')"
echo "SHA256: ${SHA}"

if command -v gh >/dev/null 2>&1; then
  gh release view "v${VERSION}" --repo liljemery/getbundled.agent >/dev/null 2>&1 \
    && gh release upload "v${VERSION}" "$OUT" --repo liljemery/getbundled.agent --clobber \
    || gh release create "v${VERSION}" "$OUT" --repo liljemery/getbundled.agent \
         --title "v${VERSION}" --notes "SHA256: \`${SHA}\`"
  echo "GitHub: https://github.com/liljemery/getbundled.agent/releases/tag/v${VERSION}"
else
  echo "gh not found — binary at ${OUT}"
fi

cat <<EOF

Next: register in backend DB (alembic or SQL):
  version=${VERSION}
  sha256=${SHA}
  download_url=https://github.com/liljemery/getbundled.agent/releases/download/v${VERSION}/${ARTIFACT}
EOF
