# AutoGet Organizer - multi-stage container build.
#
# Stage 1 builds a fully static binary with the official Go toolchain
# (CGO_ENABLED=0), trimmed with -s -w to keep the final image tiny.
# Stage 2 ships the binary on a minimal Alpine runtime under a non-root user;
# the resulting image stays well below 30MB.

# ---- Build stage -----------------------------------------------------------
FROM golang:1.27.1-alpine AS builder

WORKDIR /src

# Cache module downloads independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/organizer ./cmd/server

# ---- Runtime stage ---------------------------------------------------------
FROM alpine:3.20

# Non-root user for runtime safety; no home dir needed by the service.
RUN addgroup -g 1000 organizer \
    && adduser -u 1000 -G organizer -D -H -s /sbin/nologin organizer \
    && mkdir -p /mnt \
    && chown -R organizer:organizer /mnt

COPY --from=builder /out/organizer /usr/local/bin/organizer

# DOWNLOAD_COMPLETED_DIR / TARGET_DIR and the rest of the configuration are
# expected to be provided via environment variables (docker run -e ...).
USER organizer

EXPOSE 8080

ENTRYPOINT ["organizer"]
