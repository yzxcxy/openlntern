# Local Development

## Start external dependencies

```bash
docker compose up -d
docker compose ps
```

This Compose stack manages external dependencies and OpenViking:

- MySQL
- Redis
- MinIO
- OpenViking

## Start backend and frontend

1. Start `openIntern_backend` locally.
2. Start `openIntern_forentend` locally.
3. Export `OPENINTERN_OPENVIKING_MANAGED_EXTERNALLY=1` before starting `openIntern_backend`, so the backend does not try to launch a local OpenViking process.

## One-time initialization for a new environment

Before the first local start, run:

```bash
chmod +x scripts/init-dev-data.sh
./scripts/init-dev-data.sh
```

This script will:

- start `docker compose` dependencies
- create the MinIO bucket `open-intern`
- initialize the default login account

## Service endpoints

- MySQL: `127.0.0.1:3306`
- Redis: `127.0.0.1:6379`
- MinIO API: `http://127.0.0.1:9000`
- MinIO Console: `http://127.0.0.1:9001`
- OpenViking API: `http://127.0.0.1:1933`
- OpenViking Console: `http://127.0.0.1:8020`

## Notes

- The MySQL database name remains `open_intern`.
- This round only deploys MinIO and its management console.
- Existing COS upload behavior is unchanged in this round.
- The OpenViking container mounts the existing `openIntern_backend/ov.conf` for its own runtime configuration.
- openIntern now uploads knowledge-base and skill imports to OpenViking over HTTP before import, so OpenViking never reads backend-local temp paths directly.
- A shared Docker volume between `openIntern_backend` and OpenViking is not required for these import flows.
