# cofiswarm-agent-registry

Agent roster + mode configuration API (extracted from coordinator).

- Migration: Sprint 7 in [MIGRATION-SPRINTS](https://github.com/keepdevops/cofiswarm-docs/blob/main/MIGRATION-SPRINTS.md)
- SoT for agent JSON: `cofiswarm-config` — `data/agents/` is a mirror
- Legacy C++: `legacy/cpp/`

## HTTP (coordinator parity)

| Route | Description |
|-------|-------------|
| `GET /healthz` | Liveness |
| `GET /api/health` | `{"status":"ok","engine":"swarm-matrix"}` |
| `GET /api/agents` | Agent list |
| `PUT /api/agents/{name}/prompt` | Update system prompt (live; config mirror sprint 13) |
| `GET /api/modes` | Mode catalog + active flag |
| `GET/POST /api/modes/active` | Read/set active mode |
| `GET/PUT /api/modes/{name}/agents` | Per-mode roster |

Default listen: `:8012`.

## Build & run

```bash
make build
./bin/cofiswarm-agent-registry -swarm-config /etc/cofiswarm/config/swarm-config.json
```

## FHS

| Path | Purpose |
|------|---------|
| `/etc/cofiswarm/config/swarm-config.json` | roster source (read) |
| `/var/lib/cofiswarm/agent-registry/overrides.json` | mode roster overrides |
