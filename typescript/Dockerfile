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
EXPOSE 4000
CMD ["node", "src/server/index.ts"]
