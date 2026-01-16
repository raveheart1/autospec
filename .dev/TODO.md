# Development TODO

## Bugs

### Spec updater reformats YAML indentation
- **Issue**: When the spec updater marks a spec as completed, it re-serializes the entire YAML with different indentation (2-space → 4-space), causing massive git diffs even though content is unchanged.
- **Impact**: Makes git history noisy; hard to see actual changes.
- **Fix**: Preserve original indentation style when updating spec files, or only modify the specific fields (status, completed_at) without full re-serialization.
- **Discovered**: 2025-12-21 (spec 072-update-check-cmd)
