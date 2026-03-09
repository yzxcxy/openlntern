#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="${ROOT_DIR}/openIntern_forentend"
BACKEND_DIR="${ROOT_DIR}/openIntern_backend"

# print_help explains how to run this launcher script.
print_help() {
  cat <<'EOF'
Usage:
  ./scripts/start-dev-services.sh [auto|terminal|tmux]

Modes:
  auto      Prefer macOS Terminal windows, fallback to tmux.
  terminal  Force macOS Terminal windows (requires osascript).
  tmux      Force tmux split windows.
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

# launch_in_terminal opens three separate macOS Terminal windows.
launch_in_terminal() {
  require_command osascript

  local frontend_cmd backend_cmd viking_cmd
  frontend_cmd="cd \"${FRONTEND_DIR}\" && pnpm dev"
  backend_cmd="cd \"${BACKEND_DIR}\" && go run main.go"
  viking_cmd="openviking-server"

  local frontend_escaped backend_escaped viking_escaped
  frontend_escaped="$(escape_applescript "${frontend_cmd}")"
  backend_escaped="$(escape_applescript "${backend_cmd}")"
  viking_escaped="$(escape_applescript "${viking_cmd}")"

  osascript <<EOF
tell application "Terminal"
  activate
  do script "${frontend_escaped}"
  do script "${backend_escaped}"
  do script "${viking_escaped}"
end tell
EOF
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

  tmux new-session -d -s "${session_name}" -c "${FRONTEND_DIR}" "pnpm dev"
  tmux split-window -h -t "${session_name}:0" -c "${BACKEND_DIR}" "go run main.go"
  tmux split-window -v -t "${session_name}:0.1" -c "${ROOT_DIR}" "openviking-server"
  tmux select-layout -t "${session_name}:0" tiled
  tmux select-pane -t "${session_name}:0.0"
  tmux attach-session -t "${session_name}"
}

main() {
  local mode="${1:-auto}"

  if [[ "${mode}" == "-h" || "${mode}" == "--help" ]]; then
    print_help
    exit 0
  fi

  require_command pnpm
  require_command go
  require_command openviking-server

  case "${mode}" in
    auto)
      if [[ "$(uname -s)" == "Darwin" ]] && command -v osascript >/dev/null 2>&1; then
        launch_in_terminal
      elif command -v tmux >/dev/null 2>&1; then
        launch_in_tmux
      else
        echo "No Terminal automation or tmux available."
        echo "Please install tmux or run services manually:"
        echo "  cd \"${FRONTEND_DIR}\" && pnpm dev"
        echo "  cd \"${BACKEND_DIR}\" && go run main.go"
        echo "  openviking-server"
        exit 1
      fi
      ;;
    terminal)
      launch_in_terminal
      ;;
    tmux)
      launch_in_tmux
      ;;
    *)
      echo "Unknown mode: ${mode}" >&2
      print_help
      exit 1
      ;;
  esac
}

main "$@"
