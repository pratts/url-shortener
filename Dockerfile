# syntax=docker/dockerfile:1
#
# One build, two runtime targets:
#   docker build --target admin -t shortener-admin .
#   docker build --target redirect -t shortener-redirect .
# Base images are pinned by digest; Dependabot proposes updates.

FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
# Static, reproducible binaries without debug info.
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    for cmd in admin redirect migrate seed healthcheck; do \
      go build -trimpath -ldflags="-s -w" -o /out/$cmd ./cmd/$cmd || exit 1; \
    done

# Distroless: no shell or package manager; runs as uid 65532.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime
WORKDIR /app
COPY --from=build /out/healthcheck ./
USER nonroot:nonroot

FROM runtime AS redirect
COPY --from=build /out/redirect ./
EXPOSE 8085
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/healthcheck", "REDIRECT_PORT", "8085"]
ENTRYPOINT ["/app/redirect"]

# The admin image also carries migrate and seed, so one image can run
# migrations as a release step and create users when registration is off.
FROM runtime AS admin
COPY --from=build /out/admin /out/migrate /out/seed ./
EXPOSE 8086
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/healthcheck", "ADMIN_PORT", "8086"]
ENTRYPOINT ["/app/admin"]
