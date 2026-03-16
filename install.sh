#!/bin/sh

set -eu

REPO="go2engle/gantry"
API_URL="https://api.github.com/repos/$REPO/releases/latest"

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  log "install.sh: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

detect_os() {
  case "$(uname -s)" in
    Linux)
      printf 'linux\n'
      ;;
    Darwin)
      printf 'darwin\n'
      ;;
    *)
      fail "unsupported operating system: $(uname -s)"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64\n'
      ;;
    arm64|aarch64)
      printf 'arm64\n'
      ;;
    *)
      fail "unsupported architecture: $(uname -m)"
      ;;
  esac
}

mktemp_dir() {
  mktemp -d 2>/dev/null || mktemp -d -t gantry-install
}

latest_tag() {
  json="$(curl -fsSL "$API_URL")"
  tag="$(printf '%s' "$json" | tr -d '\n' | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  [ -n "$tag" ] || fail "failed to determine the latest Gantry release tag"
  printf '%s\n' "$tag"
}

normalize_tag() {
  tag="${GANTRY_VERSION:-}"
  if [ -z "$tag" ]; then
    latest_tag
    return
  fi

  case "$tag" in
    v*)
      printf '%s\n' "$tag"
      ;;
    *)
      printf 'v%s\n' "$tag"
      ;;
  esac
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{ print $1 }'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
    return
  fi
  fail "missing checksum tool: install sha256sum or shasum"
}

has_flag() {
  flag="$1"
  shift
  for arg in "$@"; do
    case "$arg" in
      "$flag"|"$flag"=*)
        return 0
        ;;
    esac
  done
  return 1
}

need_cmd curl
need_cmd tar
need_cmd awk
need_cmd sed
need_cmd tr

os="$(detect_os)"
arch="$(detect_arch)"
tag="$(normalize_tag)"
version="${tag#v}"
archive_name="gantry_${version}_${os}_${arch}.tar.gz"
archive_url="https://github.com/$REPO/releases/download/$tag/$archive_name"
checksums_url="https://github.com/$REPO/releases/download/$tag/checksums.txt"

tmpdir="$(mktemp_dir)"
archive_path="$tmpdir/$archive_name"
checksums_path="$tmpdir/checksums.txt"
binary_path="$tmpdir/gantry"
password_file=""
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

log "Downloading $archive_name..."
curl -fsSL "$archive_url" -o "$archive_path"
curl -fsSL "$checksums_url" -o "$checksums_path"

expected_checksum="$(awk -v file="$archive_name" '$2 == file { print $1; exit }' "$checksums_path")"
[ -n "$expected_checksum" ] || fail "checksum not found for $archive_name"
actual_checksum="$(sha256_file "$archive_path")"
[ "$actual_checksum" = "$expected_checksum" ] || fail "checksum verification failed for $archive_name"
log "Checksum verified."

tar -xzf "$archive_path" -C "$tmpdir"
[ -x "$binary_path" ] || fail "expected gantry binary was not found in $archive_name"

if ! has_flag "--admin-password-file" "$@" && ! has_flag "--admin-password-stdin" "$@" && [ -n "${GANTRY_ADMIN_PASSWORD:-}" ]; then
  password_file="$tmpdir/admin-password"
  umask 077
  printf '%s\n' "$GANTRY_ADMIN_PASSWORD" > "$password_file"
  set -- --admin-password-file "$password_file" "$@"
fi

use_tty=0
if ! has_flag "--admin-password-file" "$@" && ! has_flag "--admin-password-stdin" "$@" && [ ! -t 0 ] && [ -r /dev/tty ]; then
  use_tty=1
fi

log "Running gantry install $*"
if [ "$(id -u)" -eq 0 ]; then
  if [ "$use_tty" -eq 1 ]; then
    "$binary_path" install "$@" </dev/tty
  else
    "$binary_path" install "$@"
  fi
else
  command -v sudo >/dev/null 2>&1 || fail "sudo is required when not running as root"
  if [ "$use_tty" -eq 1 ]; then
    sudo "$binary_path" install "$@" </dev/tty
  else
    sudo "$binary_path" install "$@"
  fi
fi
