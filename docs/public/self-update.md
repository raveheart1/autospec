# Self-Update Feature

autospec includes built-in version checking and self-update functionality to help you stay up-to-date with the latest releases.

## Checking for Updates

When you run `autospec version`, the command automatically checks GitHub for newer versions:

```bash
autospec version
```

Output example (when update is available):

```
                   ▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀
                   █▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄

                   Spec-Driven Development Automation

              ╭──────────────────────────────────────────╮
              │                                          │
              │        Version    0.6.0                  │
              │         Commit    abc12345               │
              │          Built    2025-12-20             │
              │             Go    go1.23.0               │
              │       Platform    linux/amd64            │
              │                                          │
              ╰──────────────────────────────────────────╯

→ A new version is available: v0.7.0 (run 'autospec update' to upgrade)
```

### Non-Blocking Check

The version check is performed asynchronously:

- Version information displays immediately (within 100ms)
- Update notification appears afterward if the check completes in time
- Slow or failed network connections don't delay the version output
- The check times out after 500ms to prevent blocking

### Plain Output

For scripts, use `--plain` to get machine-readable output without the update check:

```bash
autospec version --plain
```

## Updating autospec

To update autospec to the latest version:

```bash
autospec update
```

The update command:

1. **Checks for updates** - Verifies if a newer version is available
2. **Downloads the binary** - Fetches the appropriate binary for your platform
3. **Verifies checksum** - Validates the download using SHA256 checksums
4. **Creates backup** - Backs up your current binary as `.bak`
5. **Installs update** - Replaces the current binary with the new version

### Example Output

```
→ Checking for updates...
→ New version available: v0.6.0 → v0.7.0
→ Downloading autospec_0.7.0_Linux_x86_64.tar.gz...
  [██████████████████████████████] 100.0% (5.2 MB/5.2 MB)
→ Verifying checksum...
✓ Checksum verified
→ Extracting binary...
→ Installing update...
✓ Successfully updated to v0.7.0
  Run 'autospec version' to verify the update.
```

## Supported Platforms

The self-update feature supports:

- **Linux** (x86_64, arm64)
- **macOS** (x86_64, arm64)

Windows users should use WSL.

## Checksum Verification

Every update is verified using SHA256 checksums:

- Checksums are fetched from the `checksums.txt` file in each release
- If checksum verification fails, the update is aborted
- Your existing binary remains intact if verification fails

## Backup and Rollback

During update:

- Your current binary is renamed to `autospec.bak`
- If installation fails, the backup is automatically restored
- After successful update, the backup is cleaned up

To manually restore from backup (if needed):

```bash
mv ~/.local/bin/autospec.bak ~/.local/bin/autospec
```

## Troubleshooting

### "cannot update dev builds"

Development builds cannot be updated automatically. If you're running a dev build:

```bash
# Check if running dev build
autospec version --plain | head -1
# Output: autospec dev

# To update, either:
# 1. Install a release version from GitHub releases
# 2. Build from source with proper version tags
```

### Permission denied

If you get a permission error:

```bash
# Check where autospec is installed
which autospec

# Ensure you have write access to that directory
# You may need to use sudo or change the installation location
```

### Network errors

Update checks require network access to GitHub:

- Ensure you can reach `api.github.com`
- Check your firewall/proxy settings
- The version command will still work without network (just no update check)

### Checksum mismatch

If checksum verification fails:

1. Try running the update again (temporary network issue)
2. Check the GitHub releases page for the correct binary
3. Download and verify the binary manually

## API Rate Limiting

autospec uses the GitHub API to check for updates:

- Unauthenticated requests: 60 per hour per IP
- If rate limited, the update check silently fails (version command still works)
- Consider using fewer terminals/scripts that run version checks
