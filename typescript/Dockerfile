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
# The TypeScript app is two services: authz (4100) and documents (4000). Both run
# in this container; the documents service reaches authz on localhost:4100 (the
# default AUTHZ_URL). Only 4000 needs publishing for clients; 4100 is internal.
EXPOSE 4000 4100
CMD ["npm", "run", "start"]
