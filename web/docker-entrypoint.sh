#!/bin/sh
# Run the SvelteKit Node SSR server in the background and nginx in the foreground.
# nginx is the public face (:80); it proxies non-API routes to the Node server on
# 127.0.0.1:3000 and /api,/health to the Go backend. If nginx exits, the
# container exits; the orchestrator restarts it (and a stalled Node surfaces as
# 502s on a healthcheck).
set -e

node build &

exec nginx -g 'daemon off;'
