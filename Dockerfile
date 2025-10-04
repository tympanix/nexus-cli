# Build stage
FROM golang:1.25.1 AS builder

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with static linking and version injection
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-extldflags \"-static\" -X main.version=${VERSION}" -o nexuscli-go ./cmd/nexuscli-go

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
