#!/usr/bin/env bash
#
# Forward a local port to rtorrent's SCGI socket on the server, run
# rtorrent-sync against it and stop the forwarding on exit.
#
# Extra arguments are passed through to rtorrent-sync.

set -euo pipefail

HOST=${RTORRENT_HOST:-btc}
PORT=${RTORRENT_PORT:-1234}
SOCKET=${RTORRENT_SOCKET:-/tmp/rtorrent.sock}

control_dir=$(mktemp -d)
control_socket="$control_dir/ssh"

cleanup() {
	if [[ -S $control_socket ]]; then
		ssh -S "$control_socket" -O exit "$HOST" 2>/dev/null || true
	fi
	rm -rf "$control_dir"
}
trap cleanup EXIT

# -f backgrounds ssh only once authenticated and the forward is set up, and
# ExitOnForwardFailure makes it fail if the local port can't be bound.
ssh -f -N -M -S "$control_socket" \
	-o ExitOnForwardFailure=yes \
	-L "127.0.0.1:$PORT:$SOCKET" "$HOST"

rtorrent-sync -addr "127.0.0.1:$PORT" -host "$HOST" "$@"
