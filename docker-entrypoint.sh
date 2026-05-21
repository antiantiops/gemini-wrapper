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
  mkdir -p "${HOME}/.local/share/keyrings"
  # Create/unlock the default collection with a blank password. This is scoped to
  # the container and fixes Antigravity CLI's Secret Service error:
  # "failed to unlock correct collection '/org/freedesktop/secrets/aliases/default'".
  printf '\n' | gnome-keyring-daemon --unlock >/dev/null 2>&1 || true
  keyring_env="$(gnome-keyring-daemon --start --components=secrets)"
  keyring_status=$?
  if [ ${keyring_status} -ne 0 ]; then
    echo "Error: gnome-keyring-daemon failed with exit code ${keyring_status}" >&2
  elif [ -n "${keyring_env}" ]; then
    eval "${keyring_env}"
    export GNOME_KEYRING_CONTROL SSH_AUTH_SOCK
  fi
  printf '\n' | gnome-keyring-daemon --unlock >/dev/null 2>&1 || true
fi

# Make the same session available to manual `docker exec` login commands and
# Go subprocesses. The .env file is deliberately plain KEY=VALUE (not shell
# escaped) so the wrapper can append it directly to exec.CommandContext.Env.
{
  [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ] && printf 'DBUS_SESSION_BUS_ADDRESS=%s\n' "${DBUS_SESSION_BUS_ADDRESS}"
  [ -n "${DBUS_SESSION_BUS_PID:-}" ] && printf 'DBUS_SESSION_BUS_PID=%s\n' "${DBUS_SESSION_BUS_PID}"
  [ -n "${GNOME_KEYRING_CONTROL:-}" ] && printf 'GNOME_KEYRING_CONTROL=%s\n' "${GNOME_KEYRING_CONTROL}"
  [ -n "${SSH_AUTH_SOCK:-}" ] && printf 'SSH_AUTH_SOCK=%s\n' "${SSH_AUTH_SOCK}"
} > /tmp/antigravity-keyring.env
chmod 600 /tmp/antigravity-keyring.env

# Shell-friendly helper for manual docker exec usage:
#   . /tmp/antigravity-keyring.sh && agy --print "hello"
sed 's/^/export /' /tmp/antigravity-keyring.env > /tmp/antigravity-keyring.sh
chmod 600 /tmp/antigravity-keyring.sh

exec "$@"
