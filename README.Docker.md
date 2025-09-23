# Docker usage

This README describes how to build and run **this service** with the provided `Dockerfile`.

## Image layout (from Dockerfile)
- Final image: based on **Alpine**.
- Binary path: **/app/server** (built with `CGO_ENABLED=0`).
- Config directory copied into the image: **/app/config**.
- Default config path via env: **CONFIG_PATH=/app/config/config.yaml**.
- Exposed port: **8080**.
- Runs as non‑root user (`appuser`, UID 10001).

---

## Quick start (local)

Build a local image:
```bash
docker build -t crypto-rate-service:local .
```

Run the container (port 8080):
```bash
docker run --rm -p 8080:8080 crypto-rate-service:local
```

If you want to provide your own config from the host:
```bash
docker run --rm -p 8080:8080 \
  -v "$(pwd)/config:/app/config:ro" \
  -e CONFIG_PATH=/app/config/config.yaml \
  crypto-rate-service:local
```

> Note: The binary expects to listen on port **8080** (as exposed by the image).

---

## Docker Compose

Create a minimal `docker-compose.yml`:
```yaml
services:
  app:
    build: .
    # or: image: crypto-rate-service:local
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config:ro
    environment:
      CONFIG_PATH: /app/config/config.yaml
```

Bring it up:
```bash
docker compose up --build
```

---

## Building for a different platform

If your host CPU differs from the target (e.g. Apple Silicon → amd64), use Buildx:
```bash
docker buildx create --use --name xbuilder || true
docker buildx build --platform linux/amd64 -t crypto-rate-service:amd64 --load .
```

Then run:
```bash
docker run --rm -p 8080:8080 crypto-rate-service:amd64
```

---

## Troubleshooting

- **Cannot bind to port**: Check nothing is already listening on `localhost:8080`.
- **Config not applied**: Ensure `-e CONFIG_PATH=/app/config/config.yaml` matches the file you mount.
- **Permission issues on volume**: The container runs as a non‑root user; ensure mounted files are world‑readable (`chmod a+r`).

---

## References
- [Docker's Go guide](https://docs.docker.com/language/golang/)
- [Buildx documentation](https://docs.docker.com/build/buildx/)