#!/bin/bash
# Control headless Spotify + NowPlayingBridge

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_FILE="$HOME/.config/headless-spotify/config"

# Load config or use defaults
if [ -f "$CONFIG_FILE" ]; then
  source "$CONFIG_FILE"
fi

LIBRESPOT_BIN="${LIBRESPOT_BIN:-$SCRIPT_DIR/go-librespot/go-librespot}"
LIBRESPOT_DIR="${LIBRESPOT_DIR:-$SCRIPT_DIR/go-librespot}"
BRIDGE_BIN="${BRIDGE_BIN:-$SCRIPT_DIR/NowPlayingBridge/NowPlayingBridge}"
API_PORT="${API_PORT:-3678}"
API_URL="http://localhost:$API_PORT"

_status_json() {
  curl -s --connect-timeout 2 "$API_URL/status" 2>/dev/null
}

_print_track() {
  python3 -c "
import json,sys
d=json.load(sys.stdin)
t=d.get('track')
if t: print(f'{t[\"name\"]} — {t[\"artist_names\"][0]}')
elif d.get('buffering'): print('Buffering...')
elif d.get('stopped'): print('Ready (nothing playing)')
else: print('Idle')
" 2>/dev/null
}

case "$1" in
  start)
    if pgrep -x go-librespot > /dev/null; then
      echo "Already running"
      _status_json | _print_track
      exit 0
    fi

    if [ ! -f "$LIBRESPOT_BIN" ]; then
      echo "Error: go-librespot not found at $LIBRESPOT_BIN"
      echo "Run the install steps in the README first, or set LIBRESPOT_BIN in $CONFIG_FILE"
      exit 1
    fi
    if [ ! -f "$BRIDGE_BIN" ]; then
      echo "Error: NowPlayingBridge not found at $BRIDGE_BIN"
      echo "Build it first: cd NowPlayingBridge && swiftc -O -framework MediaPlayer -framework AppKit -o NowPlayingBridge Sources/main.swift"
      exit 1
    fi

    cd "$LIBRESPOT_DIR" && "$LIBRESPOT_BIN" > /tmp/go-librespot.log 2>&1 &
    sleep 2

    if ! pgrep -x go-librespot > /dev/null; then
      echo "Error: go-librespot failed to start. Check /tmp/go-librespot.log"
      exit 1
    fi

    "$BRIDGE_BIN" > /tmp/nowplayingbridge.log 2>&1 &
    echo "Started"
    ;;
  stop)
    pkill -x NowPlayingBridge 2>/dev/null
    pkill -x go-librespot 2>/dev/null
    echo "Stopped"
    ;;
  status)
    if ! pgrep -x go-librespot > /dev/null; then
      echo "Not running"
      exit 1
    fi
    _status_json | _print_track
    ;;
  *)
    echo "Usage: spotify-ctl {start|stop|status}"
    exit 1
    ;;
esac
