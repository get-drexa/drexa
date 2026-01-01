# Integration tests (Go + testcontainers)

## Requirements

- A working Docker daemon reachable from the devcontainer.
  - Recommended: `.devcontainer/devcontainer.json` includes `docker-outside-of-docker`, which uses the host/Codespaces Docker.
  - Alternative: switch to `docker-in-docker` if you don’t have host Docker access.

## Run

From `apps/backend`:

```bash
go test -tags=integration ./internal/drexa -run TestRegistrationFlow -count=1
```

Notes:

- The first run pulls `postgres:16-alpine` (needs network access).
- Tests are tagged `integration` so they never run in normal `go test ./...` by accident.
- If Docker isn’t available, the test skips (rather than failing).
