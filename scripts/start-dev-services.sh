#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="${ROOT_DIR}/openIntern_forentend"
BACKEND_DIR="${ROOT_DIR}/openIntern_backend"

# print_help explains how to run this launcher script.
print_help() {
  cat <<'EOF'
Usage:
  ./scripts/start-dev-services.sh [auto|terminal|tmux] [--with-openviking|--without-openviking]

Modes:
  auto      Prefer macOS Terminal windows, fallback to tmux.
  terminal  Force macOS Terminal windows (requires osascript).
  tmux      Force tmux split windows.

OpenViking:
  --with-openviking     Require launching openviking-server.
  --without-openviking  Skip launching openviking-server.
                        Default: auto-detect (launch only when installed).
EOF
}

# require_command ensures required commands exist before launch.
require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}" >&2
    exit 1
  fi
}

# escape_applescript escapes a shell command for AppleScript string usage.
escape_applescript() {
  local raw="$1"
  raw="${raw//\\/\\\\}"
  raw="${raw//\"/\\\"}"
  printf '%s' "${raw}"
}

# print_openviking_missing_hint explains how to install the OpenViking CLI/server.
print_openviking_missing_hint() {
  cat <<'EOF'
[warn] openviking-server not found, skipping OpenViking service.
Install:
  pip install openviking
Then re-open your terminal and verify:
  openviking --help
  openviking-server --help
EOF
}

# launch_in_terminal opens three separate macOS Terminal windows.
launch_in_terminal() {
  require_command osascript

  local frontend_cmd backend_cmd
  frontend_cmd="cd \"${FRONTEND_DIR}\" && pnpm dev"
  backend_cmd="cd \"${BACKEND_DIR}\" && go run main.go"

  local frontend_escaped backend_escaped
  frontend_escaped="$(escape_applescript "${frontend_cmd}")"
  backend_escaped="$(escape_applescript "${backend_cmd}")"

  local osa_script
  read -r -d '' osa_script <<EOF || true
tell application "Terminal"
  activate
  do script "${frontend_escaped}"
  do script "${backend_escaped}"
end tell
EOF

  if [[ "${1:-false}" == "true" ]]; then
    local viking_escaped
    viking_escaped="$(escape_applescript "openviking-server")"
    osa_script="$(printf '%s\n  do script \"%s\"\nend tell' "${osa_script%$'\nend tell'}" "${viking_escaped}")"
  fi

  osascript -e "${osa_script}"
}

# launch_in_tmux creates a tmux session with three panes for services.
launch_in_tmux() {
  require_command tmux

  local session_name="openintern-dev"
  if tmux has-session -t "${session_name}" 2>/dev/null; then
    echo "tmux session '${session_name}' already exists. Attaching..."
    tmux attach-session -t "${session_name}"
    return
  fi

  local with_openviking="${1:-false}"

  tmux new-session -d -s "${session_name}" -c "${FRONTEND_DIR}" "pnpm dev"
  tmux split-window -h -t "${session_name}:0" -c "${BACKEND_DIR}" "go run main.go"
  if [[ "${with_openviking}" == "true" ]]; then
    tmux split-window -v -t "${session_name}:0.1" -c "${ROOT_DIR}" "openviking-server"
  fi
  tmux select-layout -t "${session_name}:0" tiled
  tmux select-pane -t "${session_name}:0.0"
  tmux attach-session -t "${session_name}"
}

# resolve_openviking_mode decides whether to launch openviking-server.
resolve_openviking_mode() {
  local mode="${1:-auto}"
  case "${mode}" in
    with)
      if command -v openviking-server >/dev/null 2>&1; then
        echo "true"
      else
        echo "[error] --with-openviking specified but openviking-server is missing." >&2
        print_openviking_missing_hint >&2
        exit 1
      fi
      ;;
    without)
      echo "false"
      ;;
    auto)
      if command -v openviking-server >/dev/null 2>&1; then
        echo "true"
      else
        print_openviking_missing_hint >&2
        echo "false"
      fi
      ;;
    *)
      echo "[error] invalid OpenViking mode: ${mode}" >&2
      exit 1
      ;;
  esac
}

main() {
  local mode="${1:-auto}"
  local openviking_mode="auto"

  if [[ "${2:-}" == "--with-openviking" ]]; then
    openviking_mode="with"
  elif [[ "${2:-}" == "--without-openviking" ]]; then
    openviking_mode="without"
  elif [[ -n "${2:-}" ]]; then
    echo "Unknown option: ${2}" >&2
    print_help
    exit 1
  fi

  if [[ "${mode}" == "-h" || "${mode}" == "--help" ]]; then
    print_help
    exit 0
  fi

  require_command pnpm
  require_command go
  local with_openviking
  with_openviking="$(resolve_openviking_mode "${openviking_mode}")"

  case "${mode}" in
    auto)
      if [[ "$(uname -s)" == "Darwin" ]] && command -v osascript >/dev/null 2>&1; then
        launch_in_terminal "${with_openviking}"
      elif command -v tmux >/dev/null 2>&1; then
        launch_in_tmux "${with_openviking}"
      else
        echo "No Terminal automation or tmux available."
        echo "Please install tmux or run services manually:"
        echo "  cd \"${FRONTEND_DIR}\" && pnpm dev"
        echo "  cd \"${BACKEND_DIR}\" && go run main.go"
        if [[ "${with_openviking}" == "true" ]]; then
          echo "  openviking-server"
        fi
        exit 1
      fi
      ;;
    terminal)
      launch_in_terminal "${with_openviking}"
      ;;
    tmux)
      launch_in_tmux "${with_openviking}"
      ;;
    *)
      echo "Unknown mode: ${mode}" >&2
      print_help
      exit 1
      ;;
  esac
}

main "$@"
