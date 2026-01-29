# Development TODO


## General

- [ ] Simplify generated _meta in each yaml artifact:
_meta:
    version: "1.0.0"
    created: "2025-12-18T08:48:33Z"
- [ ] Move default constitution.yaml filepath from
  .autospec/memory/constitution.yaml to just
  .autospec/constitution.yaml instead. Update doctor commands for it. (For lower version _meta in constitution warn/tell user how to fix with simple mv command)
- 'autospec init' command should ask user if they want global or local install.
  (global should be the default) config should have this as option - and --flag for it too.

## Bugs

### Spec updater reformats YAML indentation
- **Issue**: When the spec updater marks a spec as completed, it re-serializes the entire YAML with different indentation (2-space → 4-space), causing massive git diffs even though content is unchanged.
- **Impact**: Makes git history noisy; hard to see actual changes.
- **Fix**: Preserve original indentation style when updating spec files, or only modify the specific fields (status, completed_at) without full re-serialization.
- **Discovered**: 2025-12-21 (spec 072-update-check-cmd)
