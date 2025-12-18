# Risk Documentation

Document implementation risks in `plan.yaml` to acknowledge potential issues before coding begins.

## Schema

```yaml
risks:
  - id: "RISK-001"           # Optional, format: RISK-NNN
    risk: "Database migration may cause downtime"
    likelihood: "medium"     # low | medium | high
    impact: "high"           # low | medium | high
    mitigation: "Run migration during maintenance window"
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `risk` | Yes | Description of the risk |
| `likelihood` | Yes | Probability: `low`, `medium`, `high` |
| `impact` | Yes | Severity: `low`, `medium`, `high` |
| `mitigation` | No | Strategy to address the risk |
| `id` | No | Unique identifier (RISK-NNN format) |

## Validation

- **Errors**: Missing required fields, invalid enum values, malformed IDs
- **Warnings**: High-impact risks without mitigation (non-blocking)

```bash
# Validate plan.yaml including risks
autospec artifact specs/001-feature/plan.yaml
```

## Status Display

`autospec st` shows risk summary when plan.yaml contains risks:

```
spec: 001-dark-mode
  artifacts: [spec.yaml plan.yaml tasks.yaml]
  risks: 3 total (1 high, 2 medium)
  progress: 8/15 tasks (53%)
```

## Notes

- The `risks` section is **optional** for backward compatibility
- Empty arrays are valid: `risks: []`
- Only high-impact risks trigger mitigation warnings
