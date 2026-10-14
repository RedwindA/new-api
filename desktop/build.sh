#!/usr/bin/env bash
# Builds the New API desktop app with MyGo: the frontend, the new-api server
# for each platform, then the app and its installers in desktop/build/.
#
#   ./build.sh                                  # this machine's platform
#   ./build.sh windows/amd64                    # e.g. from Linux or macOS
#   ./build.sh windows/amd64,linux/amd64
#
# SKIP_WEB=1 reuses web/dist from an earlier build.
set -euo pipefail

cd "$(dirname "$0")"
DESKTOP_DIR=$(pwd)
ROOT_DIR=$(cd .. && pwd)
PLATFORMS=${1:-$(go env GOOS)/$(go env GOARCH)}

VERSION=$(git -C "$ROOT_DIR" describe --tags --match 'v[0-9]*' 2>/dev/null || echo v0.0.0)
export NEW_API_VERSION=${VERSION#v}

if [[ "${SKIP_WEB:-}" != "1" ]]; then
  echo "==> Building frontend"
  (cd "$ROOT_DIR/web" && bun install --frozen-lockfile && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION="$VERSION" bun run build)
fi

# go list reports the directory of a module only once it is downloaded.
(cd "$ROOT_DIR" && go mod download github.com/Calcium-Ion/moejs)
PGO="$(cd "$ROOT_DIR" && go list -m -f '{{.Dir}}' github.com/Calcium-Ion/moejs)/default.pgo"

build_server() {
  local goos=$1 goarch=$2 exe=new-api
  [[ "$goos" == "windows" ]] && exe=new-api.exe
  local out="$DESKTOP_DIR/resources/$goos-$goarch/bin/$exe"
  echo "==> Building new-api for $goos/$goarch"
  (cd "$ROOT_DIR" && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -pgo="$PGO" \
    -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$VERSION'" -o "$out")
}

# Each run ships only the servers of the platforms it builds.
rm -rf "$DESKTOP_DIR"/resources/*-*/bin
IFS=',' read -ra TARGETS <<< "$PLATFORMS"
for target in "${TARGETS[@]}"; do
  goos=${target%/*}
  goarch=${target#*/}
  if [[ "$goarch" == "universal" ]]; then
    build_server "$goos" arm64
    build_server "$goos" amd64
  else
    build_server "$goos" "$goarch"
  fi
done

mkdir -p resources/licenses
cp "$ROOT_DIR/LICENSE" "$ROOT_DIR/NOTICE" "$ROOT_DIR/THIRD-PARTY-LICENSES.md" resources/licenses/

echo "==> Building the app for $PLATFORMS"
go tool mygo build -platform "$PLATFORMS"

echo "Done: $DESKTOP_DIR/build"
