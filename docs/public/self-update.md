# Self-Update Feature

autospec includes built-in version checking and self-update functionality to help you stay up-to-date with the latest releases.

## Checking for Updates

Use `autospec check` (or the short alias `autospec ck`) to check for newer versions:

```bash
autospec check
autospec ck       # Short alias
```

Output example (when update is available):

```
Current version: v0.6.0
Latest version:  v0.7.0

→ A new version is available! Run 'autospec update' to upgrade.
```

### Plain Output

For scripts, use `--plain` to get machine-readable output:

```bash
autospec ck --plain
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
6. **Syncs configuration** - Adds new config options and removes deprecated ones

### Configuration Sync

After a successful update, your user configuration (`~/.config/autospec/config.yml`) is automatically synchronized:

- **New options** are added with their default values
- **Deprecated options** are removed
- **Your existing settings** are always preserved

You can also manually sync your config at any time:

```bash
autospec config sync --dry-run    # Preview changes
autospec config sync              # Apply changes
```

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
→ Syncing configuration...
✓ Config synced: 2 new options added, 1 deprecated removed
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

### Checksum mismatch

If checksum verification fails:

1. Try running the update again (temporary network issue)
2. Check the GitHub releases page for the correct binary
3. Download and verify the binary manually

## API Rate Limiting

autospec uses the GitHub API to check for updates:

- Unauthenticated requests: 60 per hour per IP
- If rate limited, the check command will report an error
