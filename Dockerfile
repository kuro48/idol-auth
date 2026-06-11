FROM golang:1.26.2-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates

FROM base AS deps
COPY go.mod go.sum ./
RUN go mod download

FROM node:22-alpine AS build-docs
WORKDIR /docs
COPY docs-site/package.json docs-site/package-lock.json ./
RUN npm ci
COPY docs-site/vite.config.mjs ./
COPY docs-site/index.html ./
COPY docs-site/src ./src
COPY docs-site/public ./public
COPY docs-site/start ./start
COPY docs-site/concepts ./concepts
COPY docs-site/sdk ./sdk
COPY docs-site/management ./management
COPY docs-site/account ./account
COPY docs-site/security ./security
COPY docs-site/api ./api
COPY docs/swagger/swagger.json ./public/openapi.json
RUN npm run build

FROM deps AS build-app
COPY . .
COPY --from=build-docs /docs/dist ./internal/docsfs/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

FROM deps AS build-migrate
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/migrate ./cmd/migrate

FROM deps AS build-configcheck
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/configcheck ./cmd/configcheck

FROM deps AS build-demo
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/demo ./cmd/demo

FROM deps AS build-portal
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/portal ./cmd/portal

FROM deps AS build-adminctl
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/adminctl ./cmd/adminctl

FROM alpine:3.21 AS app
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S idol-auth \
    && adduser -S -G idol-auth -H -h /nonexistent idol-auth \
    && mkdir -p /var/lib/idol-auth/uploads \
    && chown -R idol-auth:idol-auth /var/lib/idol-auth
COPY --from=build-app /out/server /server
COPY --from=build-app /app/internal/infra/db/migrations /migrations
USER idol-auth
ENTRYPOINT ["/server"]

FROM gcr.io/distroless/static-debian12 AS migrate
COPY --from=build-migrate /out/migrate /migrate
COPY --from=build-migrate /app/internal/infra/db/migrations /migrations
ENTRYPOINT ["/migrate"]

FROM gcr.io/distroless/static-debian12 AS configcheck
COPY --from=build-configcheck /out/configcheck /configcheck
ENTRYPOINT ["/configcheck"]

FROM gcr.io/distroless/static-debian12 AS demo
COPY --from=build-demo /out/demo /demo
ENTRYPOINT ["/demo"]

FROM gcr.io/distroless/static-debian12 AS portal
COPY --from=build-portal /out/portal /portal
ENTRYPOINT ["/portal"]

FROM gcr.io/distroless/static-debian12 AS adminctl
COPY --from=build-adminctl /out/adminctl /adminctl
ENTRYPOINT ["/adminctl"]
