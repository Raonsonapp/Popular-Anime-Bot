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

SESSION_FILE="${SESSION_NAME:-/data/userbot.session}"

if [ -n "$TELEGRAM_API_ID" ] && [ -n "$TELEGRAM_API_HASH" ]; then
    (
        # PORT is unset here so neither the listener nor the login helper
        # try to bind it - only the Go binary below needs to satisfy
        # Render's port check.
        cd /app/listener
        unset PORT
        while true; do
            if [ -f "$SESSION_FILE" ]; then
                python3 main.py || true
                echo "[entrypoint] mtproto-listener exited, restarting in 5s..." >&2
            else
                # No session yet - client.start() would otherwise prompt
                # for a phone number/code interactively, which has no
                # terminal to read from here. Render's free tier has no
                # Shell access either, so serve a tiny web-based login
                # flow instead (see docs/RENDER.md) - once it succeeds it
                # writes the session file and exits, and the next loop
                # iteration starts the real listener automatically.
                echo "[entrypoint] no session file at $SESSION_FILE yet - complete login at https://<this-service>/telegram-login/?key=<INTERNAL_API_KEY>" >&2
                python3 weblogin_server.py || true
                echo "[entrypoint] web login helper exited, re-checking in 5s..." >&2
            fi
            sleep 5
        done
    ) &
else
    echo "[entrypoint] TELEGRAM_API_ID/TELEGRAM_API_HASH not set, skipping mtproto-listener" >&2
fi

exec /usr/local/bin/api
