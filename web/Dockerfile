# syntax=docker/dockerfile:1
#
# ilabhu web — Next.js 16 standalone image.
#
# Multi-stage so the runtime image carries only the standalone server.js
# bundle and the production node_modules subset Next emits.

FROM node:22-bookworm-slim AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM node:22-bookworm-slim AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-bookworm-slim AS runtime
WORKDIR /app
ENV NODE_ENV=production \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# next.config.ts sets `output: "standalone"`, which produces a slim
# server.js + the bare-minimum node_modules under .next/standalone/.
COPY --from=build /app/public ./public
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static

USER node
EXPOSE 3000
CMD ["node", "server.js"]
