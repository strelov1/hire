# Production image for the freehire web frontend (SvelteKit SSR, adapter-node).
#
# A single public container keeps everything same-origin for the SameSite=Lax
# auth cookie: nginx listens on :80 and routes
#   - /api, /health  -> the Go backend (app:8080)
#   - everything else -> the Node SSR server (127.0.0.1:3000)
# The Node server fetches the Go API server-to-server via API_INTERNAL_URL.
#
# Build context is the web/ directory:
#   docker build -f web/Dockerfile web/

# --- build stage ---
FROM node:22-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build
# Keep only production deps for the runtime image (the build tools are no longer
# needed; adapter-node has already bundled the app).
RUN npm prune --omit=dev

# --- runtime stage ---
FROM node:22-alpine
RUN apk add --no-cache nginx
WORKDIR /app
COPY --from=build /src/build ./build
COPY --from=build /src/node_modules ./node_modules
COPY --from=build /src/package.json ./package.json
# The CLI install script is served as a static file by nginx (see nginx.conf).
COPY --from=build /src/public/install.sh /usr/share/nginx/html/install.sh
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY docker-entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Node SSR server settings. ORIGIN must be set at deploy to the public site URL
# (used for canonical/sitemap/JSON-LD); PROTOCOL_HEADER/HOST_HEADER let it derive
# the origin from nginx's forwarded headers as a fallback. See web/.env.example.
ENV PORT=3000
ENV API_INTERNAL_URL=http://app:8080
ENV PROTOCOL_HEADER=x-forwarded-proto
ENV HOST_HEADER=x-forwarded-host

EXPOSE 80
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
