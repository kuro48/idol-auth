FROM golang:1.23-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates

FROM base AS deps
COPY go.mod go.sum ./
RUN go mod download

FROM deps AS build-app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

FROM deps AS build-migrate
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12 AS app
COPY --from=build-app /out/server /server
COPY --from=build-app /app/internal/infra/db/migrations /migrations
ENTRYPOINT ["/server"]

FROM gcr.io/distroless/static-debian12 AS migrate
COPY --from=build-migrate /out/migrate /migrate
COPY --from=build-migrate /app/internal/infra/db/migrations /migrations
ENTRYPOINT ["/migrate"]
