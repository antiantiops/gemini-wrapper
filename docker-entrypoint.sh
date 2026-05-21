#!/usr/bin/env bash
set -euo pipefail

# Antigravity CLI stores OAuth tokens in the system keyring. Minimal Docker
# images do not provide a DBus session/Secret Service by default, so start one
# for the wrapper process and every agy subprocess it spawns.
if command -v dbus-launch >/dev/null 2>&1; then
  eval "$(dbus-launch --sh-syntax)"
  export DBUS_SESSION_BUS_ADDRESS DBUS_SESSION_BUS_PID
fi

if command -v gnome-keyring-daemon >/dev/null 2>&1; then
  keyring_env="$(gnome-keyring-daemon --start --components=secrets 2>/dev/null || true)"
  if [ -n "${keyring_env}" ]; then
    eval "${keyring_env}"
    export GNOME_KEYRING_CONTROL SSH_AUTH_SOCK
  fi
fi

# Make the same session available to manual `docker exec` login commands:
#   . /tmp/antigravity-keyring.env && agy --print "hello"
{
  [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ] && printf 'export DBUS_SESSION_BUS_ADDRESS=%q\n' "${DBUS_SESSION_BUS_ADDRESS}"
  [ -n "${DBUS_SESSION_BUS_PID:-}" ] && printf 'export DBUS_SESSION_BUS_PID=%q\n' "${DBUS_SESSION_BUS_PID}"
  [ -n "${GNOME_KEYRING_CONTROL:-}" ] && printf 'export GNOME_KEYRING_CONTROL=%q\n' "${GNOME_KEYRING_CONTROL}"
  [ -n "${SSH_AUTH_SOCK:-}" ] && printf 'export SSH_AUTH_SOCK=%q\n' "${SSH_AUTH_SOCK}"
} > /tmp/antigravity-keyring.env
chmod 600 /tmp/antigravity-keyring.env

exec "$@"
