FROM golang:1.25-alpine AS dev
WORKDIR /workspace

FROM dev AS build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bin/server ./cmd/server

FROM alpine:3.20 AS runtime
WORKDIR /app
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /workspace/bin/server ./server
EXPOSE 4001
CMD ["./server"]
