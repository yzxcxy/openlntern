# Local Development

## Start external dependencies

```bash
docker compose up -d
docker compose ps
```

This Compose stack only manages external dependencies:

- MySQL
- Redis
- MinIO

## Start backend and frontend

1. Start `openIntern_backend` locally.
2. Start `openIntern_forentend` locally.
3. Keep `openviking` local. It is still managed by the backend process and is not part of Compose in this round.

## Service endpoints

- MySQL: `127.0.0.1:3306`
- Redis: `127.0.0.1:6379`
- MinIO API: `http://127.0.0.1:9000`
- MinIO Console: `http://127.0.0.1:9001`

## Notes

- The MySQL database name remains `open_intern`.
- This round only deploys MinIO and its management console.
- Existing COS upload behavior is unchanged in this round.
- `openviking` remains local for now and will be migrated later if needed.
