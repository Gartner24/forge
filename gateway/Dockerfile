FROM rust:latest

WORKDIR /opt/infra/forge/gateway

# Copy sources
COPY Cargo.toml Cargo.lock ./
COPY src ./src

# Build in release mode
RUN cargo build --release

# Create a non-root user for the gateway
RUN useradd -m -u 1000 gateway \
 && cp target/release/forge-gateway /usr/local/bin/forge-gateway \
 && rm -rf target

# The gateway listens on 2224 inside the container.
EXPOSE 2224

USER gateway

ENTRYPOINT ["/usr/local/bin/forge-gateway"]