# Build Controller
FROM public.ecr.aws/docker/library/golang:1.25.0-bookworm AS build
WORKDIR /controller

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Add the rest
COPY . .

# Download Dependencies
RUN go mod download

WORKDIR /controller

# Pure-Go static-ish build (no CGO)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /usr/local/bin/controller ./cmd/controller

# Runtime
FROM public.ecr.aws/docker/library/alpine:3.22.1

# TLS Certs
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 controller

COPY --from=build /usr/local/bin/controller /usr/local/bin/controller
RUN chmod 0755 /usr/local/bin/controller
USER controller

ENTRYPOINT ["/usr/local/bin/controller"]