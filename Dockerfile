FROM golang:1.26-alpine AS builder

WORKDIR /src

# Copy only what the CLI depends on
COPY apps/cli/ ./apps/cli/
COPY packages/core/ ./packages/core/

# Build from inside apps/cli so the replace directive (../../packages/core)
# resolves to /src/packages/core. GOWORK=off keeps other workspace modules out.
WORKDIR /src/apps/cli
RUN CGO_ENABLED=0 GOWORK=off go build -ldflags="-s -w" -o /usr/local/bin/clrsig .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /usr/local/bin/clrsig /usr/local/bin/clrsig
ENTRYPOINT ["clrsig"]
