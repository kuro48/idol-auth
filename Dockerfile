FROM golang:1.26.2-alpine AS base
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

FROM deps AS build-demo
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/demo ./cmd/demo

FROM deps AS build-portal
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/portal ./cmd/portal

FROM deps AS build-adminctl
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/adminctl ./cmd/adminctl

FROM gcr.io/distroless/static-debian12 AS app
COPY --from=build-app /out/server /server
COPY --from=build-app /app/internal/infra/db/migrations /migrations
ENTRYPOINT ["/server"]

FROM gcr.io/distroless/static-debian12 AS migrate
COPY --from=build-migrate /out/migrate /migrate
COPY --from=build-migrate /app/internal/infra/db/migrations /migrations
ENTRYPOINT ["/migrate"]

FROM gcr.io/distroless/static-debian12 AS demo
COPY --from=build-demo /out/demo /demo
ENTRYPOINT ["/demo"]

FROM gcr.io/distroless/static-debian12 AS portal
COPY --from=build-portal /out/portal /portal
ENTRYPOINT ["/portal"]

FROM gcr.io/distroless/static-debian12 AS adminctl
COPY --from=build-adminctl /out/adminctl /adminctl
ENTRYPOINT ["/adminctl"]
