FROM golang:1.26-alpine AS builder

WORKDIR /src

# Copy only what the CLI depends on
COPY apps/cli/ ./apps/cli/
COPY packages/core/ ./packages/core/

# GOWORK=off uses the replace directive in apps/cli/go.mod (../../packages/core)
# which resolves correctly to /src/packages/core relative to the go.mod file.
RUN CGO_ENABLED=0 GOWORK=off go build -ldflags="-s -w" -o /usr/local/bin/clrsig ./apps/cli

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /usr/local/bin/clrsig /usr/local/bin/clrsig
ENTRYPOINT ["clrsig"]
