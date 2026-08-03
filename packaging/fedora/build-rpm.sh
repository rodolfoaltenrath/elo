#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

version="${1:-${ELO_VERSION:-0.1.0}}"
fedora_version="${FEDORA_VERSION:-$(rpm -E '%{fedora}')}"

if [[ ! "$version" =~ ^[0-9][0-9A-Za-z.+_-]*$ ]]; then
  echo "Versão inválida para o RPM: $version" >&2
  exit 1
fi
if [[ ! "$fedora_version" =~ ^[0-9]+$ ]]; then
  echo "Versão do Fedora inválida: $fedora_version" >&2
  exit 1
fi

export ELO_VERSION="$version"
export ELO_RPM_RELEASE="${ELO_RPM_RELEASE:-1.fc${fedora_version}}"

echo "Compilando o Elo ${ELO_VERSION} no Fedora ${fedora_version}..."
npm --prefix frontend ci
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean
go test ./...

# Fedora 42+ não oferece mais o OpenJDK 8 tradicional. O runtime Temurin fica
# privado ao Elo, sem alterar o Java padrão da máquina da pessoa usuária.
temurin_version="8u492b09"
temurin_tag="jdk8u492-b09"
temurin_archive="OpenJDK8U-jre_x64_linux_hotspot_${temurin_version}.tar.gz"
temurin_base="https://github.com/adoptium/temurin8-binaries/releases/download/${temurin_tag}"
jre_stage="$repo_root/build/fedora-jre8"
temp_dir="$(mktemp -d)"
trap 'rm -rf -- "$temp_dir"' EXIT

case "$jre_stage" in
  "$repo_root"/build/*) ;;
  *) echo "Diretório de estágio inseguro: $jre_stage" >&2; exit 1 ;;
esac
rm -rf -- "$jre_stage"
mkdir -p "$jre_stage"

curl --fail --location --retry 3 \
  --output "$temp_dir/$temurin_archive" \
  "$temurin_base/$temurin_archive"
curl --fail --location --retry 3 \
  --output "$temp_dir/$temurin_archive.sha256.txt" \
  "$temurin_base/$temurin_archive.sha256.txt"
(
  cd "$temp_dir"
  sha256sum --check "$temurin_archive.sha256.txt"
)
tar --extract --gzip --file "$temp_dir/$temurin_archive" \
  --directory "$jre_stage" --strip-components=1

mkdir -p dist
go run github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.43.4 package \
  --config packaging/fedora/nfpm.yaml \
  --packager rpm \
  --target dist/

rpm_path="$(find dist -maxdepth 1 -type f -name "elo-${ELO_VERSION}-*.x86_64.rpm" -print -quit)"
if [[ -z "$rpm_path" ]]; then
  echo "O RPM não foi encontrado em dist/." >&2
  exit 1
fi

echo "RPM criado: $rpm_path"
rpm --query --package --info "$rpm_path"
rpm --query --package --requires "$rpm_path"
