# AutoGet Organizer - multi-stage container build.
#
# Stage 1 builds a fully static binary with the official Go toolchain
# (CGO_ENABLED=0), trimmed with -s -w to keep the final image tiny.
# Stage 2 ships the binary on a minimal Alpine runtime under a non-root user;
# the resulting image stays well below 30MB.

ARG GO_VERSION=1.27.1
ARG ALPINE_VERSION=latest
ARG UID=99 
ARG GID=100

# ---- Build stage -----------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

# Cache module downloads independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/organizer ./cmd/server

# ---- Runtime stage ---------------------------------------------------------
FROM alpine:${ALPINE_VERSION}

ARG UID
ARG GID

# Install ca-certificates for HTTPS/TLS requests and set up runtime user
RUN apk add --no-cache ca-certificates \
    && addgroup -g ${GID} organizer \
    && adduser -u ${UID} -G organizer -D -H -s /sbin/nologin organizer \
    && mkdir -p /mnt \
    && chown -R organizer:organizer /mnt

COPY --from=builder /out/organizer /usr/local/bin/organizer

# DOWNLOAD_COMPLETED_DIR / TARGET_DIR and the rest of the configuration are
# expected to be provided via environment variables (docker run -e ...).
USER organizer

EXPOSE 8000

ENTRYPOINT ["organizer"]
