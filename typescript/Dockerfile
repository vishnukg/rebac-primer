FROM node:22-slim AS deps
WORKDIR /workspace
COPY package.json package-lock.json ./
RUN npm ci

FROM deps AS build
COPY tsconfig.json vitest.config.ts ./
COPY src ./src
COPY test ./test
RUN npm run build

FROM node:22-slim AS runtime
WORKDIR /workspace
ENV NODE_ENV=production
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
COPY --from=build /workspace/src ./src
# The TypeScript app is two services: authz (4100) and documents (4000).
# docker-compose runs them as separate containers (overriding `command` per
# service). The default below runs both in one process — handy for a local
# `npm start` or a single-container `docker run`.
EXPOSE 4000 4100
CMD ["npm", "run", "start"]
