#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
project_dir=$(cd -- "${script_dir}/../.." && pwd)
output_dir="${project_dir}/dist"
binary="${project_dir}/build/bin/elo"
topdir=$(mktemp -d)

cleanup() {
  rm -rf -- "${topdir}"
}
trap cleanup EXIT

if ! command -v rpmbuild >/dev/null 2>&1; then
  echo "Comando obrigatório não encontrado: rpmbuild" >&2
  exit 1
fi

if [[ ! -x "${binary}" ]]; then
  for command in go npm; do
    if ! command -v "${command}" >/dev/null 2>&1; then
      echo "Comando necessário para compilar o Elo não encontrado: ${command}" >&2
      exit 1
    fi
  done
  (
    cd "${project_dir}"
    go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean
  )
fi

install -d "${topdir}/BUILD" "${topdir}/BUILDROOT" "${topdir}/RPMS" \
  "${topdir}/SOURCES" "${topdir}/SPECS" "${topdir}/SRPMS" "${output_dir}"
install -m 0755 "${binary}" "${topdir}/SOURCES/elo"
install -m 0644 "${project_dir}/packaging/arch/elo.desktop" "${topdir}/SOURCES/elo.desktop"
install -m 0644 "${project_dir}/build/appicon.png" "${topdir}/SOURCES/elo.png"
install -m 0644 "${project_dir}/README.md" "${topdir}/SOURCES/README.md"
install -m 0644 "${script_dir}/elo.spec" "${topdir}/SPECS/elo.spec"

rpmbuild \
  --define "_topdir ${topdir}" \
  --define "_rpmdir ${output_dir}" \
  -bb "${topdir}/SPECS/elo.spec"

echo "Pacote criado em ${output_dir}"
