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
- The OpenViking container mounts the existing `openIntern_backend/ov.conf` and persists its workspace with a named Docker volume.
