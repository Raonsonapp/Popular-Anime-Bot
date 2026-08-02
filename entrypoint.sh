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

if [ -z "$TELEGRAM_API_ID" ] || [ -z "$TELEGRAM_API_HASH" ]; then
    echo "[entrypoint] TELEGRAM_API_ID/TELEGRAM_API_HASH not set, skipping mtproto-listener" >&2
elif [ ! -f "$SESSION_FILE" ]; then
    # No session yet - client.start() would try to prompt for a phone
    # number/code interactively, which has no terminal to read from here
    # and just crash-loops forever. Log-in must happen once via the Shell
    # tab (`cd listener && python login.py`); the listener starts
    # automatically on the next restart/redeploy after that.
    echo "[entrypoint] no session file at $SESSION_FILE yet - run 'cd listener && python login.py' in the Shell tab, then restart this service. Skipping mtproto-listener for now." >&2
else
    (
        # PORT is unset here so the listener doesn't also try to bind it -
        # only the Go binary below needs to satisfy Render's port check.
        cd /app/listener
        unset PORT
        while true; do
            python3 main.py || true
            echo "[entrypoint] mtproto-listener exited, restarting in 5s..." >&2
            sleep 5
        done
    ) &
fi

exec /usr/local/bin/api
