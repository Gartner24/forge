FROM rust:1.78 as builder

WORKDIR /app

COPY Cargo.toml Cargo.lock ./
COPY src ./src

RUN cargo build --release

FROM debian:bookworm-slim

RUN useradd -m -u 1000 gateway

WORKDIR /opt/infra/forge/gateway

# Binary built in the first stage
COPY --from=builder /app/target/release/forge-gateway /usr/local/bin/forge-gateway

# The gateway listens on 2224 inside the container.
EXPOSE 2224

USER gateway

ENTRYPOINT ["/usr/local/bin/forge-gateway"]

