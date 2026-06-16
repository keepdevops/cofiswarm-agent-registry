# cofiswarm-agent-registry

Cofiswarm component: `agent-registry`.

- Layout: [REPO-STANDARD-LAYOUT](https://github.com/keepdevops/cofiswarmdev/blob/main/docs/REPO-STANDARD-LAYOUT.md)
- Migration: [MIGRATION-SPRINTS](https://github.com/keepdevops/cofiswarmdev/blob/main/docs/MIGRATION-SPRINTS.md)

## FHS paths

| Path | Purpose |
|------|---------|
| `/etc/cofiswarm/agent-registry/` | config |
| `/var/lib/cofiswarm/agent-registry/` | state |
| `/var/log/cofiswarm/agent-registry/` | logs |

## Test

```bash
./test/scripts/assert-layout.sh agent-registry
```
