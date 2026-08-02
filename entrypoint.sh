#!/bin/sh
# Starts the MTProto listener as a background process (auto-restarting if
# it crashes) and runs the Go API/bot binary as the container's main
# process. Both live in one container because that's the only free-tier
# option on Render - see docs/RENDER.md.
set -e

# The listener talks to the API over loopback, on whatever port Render
# assigned this container - no public URL needed since they're in the
# same process group. Overrides anything set in the dashboard.
export API_BASE_URL="http://localhost:${PORT:-8080}"

if [ -n "$TELEGRAM_API_ID" ] && [ -n "$TELEGRAM_API_HASH" ]; then
    (
        cd /app/listener
        while true; do
            python3 main.py || true
            echo "[entrypoint] mtproto-listener exited, restarting in 5s..." >&2
            sleep 5
        done
    ) &
else
    echo "[entrypoint] TELEGRAM_API_ID/TELEGRAM_API_HASH not set, skipping mtproto-listener" >&2
fi

exec /usr/local/bin/api
