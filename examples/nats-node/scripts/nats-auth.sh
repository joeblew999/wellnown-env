#!/bin/bash
# nats-auth.sh: Output NATS CLI auth arguments based on .auth/mode
#
# Usage: nats $(./scripts/nats-auth.sh) <command>
# Or source it: AUTH_ARGS=$(./scripts/nats-auth.sh)
#
# This script reads the auth mode from .auth/mode and outputs the
# appropriate CLI arguments for authentication.

AUTH_ARGS=""

if [ -f .auth/mode ]; then
  MODE=$(cat .auth/mode)
  case "$MODE" in
    token)
      [ -f .auth/token ] && AUTH_ARGS="--token $(cat .auth/token)"
      ;;
    nkey)
      [ -f .auth/user.nk ] && AUTH_ARGS="--nkey .auth/user.nk"
      ;;
    jwt)
      [ -f .auth/creds/user.creds ] && AUTH_ARGS="--creds .auth/creds/user.creds"
      ;;
  esac
fi

echo "$AUTH_ARGS"
