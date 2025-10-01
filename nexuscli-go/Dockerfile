# Build stage
FROM golang:1.22 AS builder

WORKDIR /build

# Copy all source files including vendored dependencies
COPY . .

# Build the binary with static linking using vendored dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -ldflags '-extldflags "-static"' -o nexuscli-go .

# Final stage - use scratch for minimal image
FROM scratch

# Copy ca-certificates from builder for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder
COPY --from=builder /build/nexuscli-go /nexuscli-go

# Set the binary as entrypoint
ENTRYPOINT ["/nexuscli-go"]

# Default command shows help
CMD ["--help"]
