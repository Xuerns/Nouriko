# syntax=docker/dockerfile:1

ARG NODE_VERSION=26

# ---- Base stage ----
FROM node:${NODE_VERSION}-alpine AS base
# libc6-compat is required by some native Node modules on Alpine.
RUN apk add --no-cache libc6-compat
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1 \
    PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH
# Install the exact pnpm version pinned in package.json "packageManager".
COPY package.json ./
RUN npm install -g "$(node -p "require('./package.json').packageManager")"

# ---- Dependencies stage ----
FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm install --frozen-lockfile

# ---- Build stage ----
FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Public variables are inlined into the client bundle at build time.
ARG NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_GOOGLE_CLIENT_ID
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
ENV NEXT_PUBLIC_GOOGLE_CLIENT_ID=${NEXT_PUBLIC_GOOGLE_CLIENT_ID}

RUN pnpm run build

# ---- Runtime stage ----
FROM node:${NODE_VERSION}-alpine AS runner
WORKDIR /app

# Upgrade Alpine packages and strip npm/corepack/yarn from the runtime image
# (only `node server.js` is needed; removing bundled npm eliminates unused CVEs).
RUN apk upgrade --no-cache && \
    rm -rf /usr/local/lib/node_modules \
           /usr/local/bin/npm \
           /usr/local/bin/npx \
           /usr/local/bin/corepack \
           /opt/yarn*

ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# Standalone output contains only the files needed to run the server.
COPY --from=builder --chown=node:node /app/public ./public
COPY --from=builder --chown=node:node /app/.next/standalone ./
COPY --from=builder --chown=node:node /app/.next/static ./.next/static

USER node

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["node", "-e", "fetch('http://127.0.0.1:'+process.env.PORT).then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"]

CMD ["node", "server.js"]
