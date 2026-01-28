# Manual Testing Plan: Fix Agent Fetch Hang (105-fix-agent-fetch-hang)

This document outlines manual testing steps to verify the fix for the `autospec new-feature` hang issue in sandbox/agent environments.

## Prerequisites

- Access to a git repository with at least one SSH remote configured
- Ability to unset `SSH_AUTH_SOCK` environment variable
- Built `autospec` binary (run `make build`)

## Test Cases

### Test 1: SSH Remote Skip Behavior (No SSH Agent)

**Objective**: Verify SSH remotes are skipped when SSH_AUTH_SOCK is not set.

**Steps**:
1. Open a new terminal session
2. Ensure SSH_AUTH_SOCK is unset:
   ```bash
   unset SSH_AUTH_SOCK
   echo $SSH_AUTH_SOCK  # Should be empty
   ```
3. Navigate to a git repo with SSH remotes:
   ```bash
   git remote -v  # Verify SSH URLs present (git@github.com:...)
   ```
4. Run the new-feature command:
   ```bash
   autospec new-feature "test ssh skip"
   ```

**Expected Results**:
- [ ] Command completes within 5 seconds
- [ ] No hang or timeout waiting for SSH connection
- [ ] Feature branch is created successfully
- [ ] Debug log shows SSH remotes were skipped (run with `--verbose` if needed)

**Cleanup**:
```bash
git checkout main && git branch -D <created-branch>
```

---

### Test 2: --no-fetch Flag Behavior

**Objective**: Verify --no-fetch flag skips all fetch operations.

**Steps**:
1. Open a terminal with SSH agent available:
   ```bash
   echo $SSH_AUTH_SOCK  # Should have a value
   ```
2. Run with --no-fetch flag:
   ```bash
   autospec new-feature --no-fetch "test no fetch"
   ```

**Expected Results**:
- [ ] Command completes without any network calls
- [ ] No fetch operations attempted (verify with network monitoring or verbose output)
- [ ] Feature branch is created successfully

**Cleanup**:
```bash
git checkout main && git branch -D <created-branch>
```

---

### Test 3: Timeout Behavior

**Objective**: Verify fetch operations timeout after 60 seconds.

**Steps**:
1. Configure a remote with unreachable host:
   ```bash
   git remote add timeout-test git@192.0.2.1:test/repo.git  # Non-routable IP
   ```
2. Ensure SSH agent is available:
   ```bash
   echo $SSH_AUTH_SOCK  # Should have value
   ```
3. Run new-feature command:
   ```bash
   time autospec new-feature "test timeout"
   ```

**Expected Results**:
- [ ] Command does NOT hang indefinitely
- [ ] Fetch to unreachable host times out (within ~60 seconds)
- [ ] Warning logged about fetch timeout
- [ ] Command completes and creates feature branch

**Cleanup**:
```bash
git checkout main && git branch -D <created-branch>
git remote remove timeout-test
```

---

### Test 4: Normal Operation (SSH Agent Available)

**Objective**: Verify backwards compatibility - normal fetch works with SSH agent.

**Steps**:
1. Ensure SSH agent is running with keys loaded:
   ```bash
   ssh-add -l  # Should list keys
   ```
2. Verify SSH_AUTH_SOCK is set:
   ```bash
   echo $SSH_AUTH_SOCK
   ```
3. Run new-feature command:
   ```bash
   autospec new-feature "test normal operation"
   ```

**Expected Results**:
- [ ] SSH remotes are fetched (not skipped)
- [ ] HTTPS remotes are fetched
- [ ] Feature branch is created with latest remote data
- [ ] No regression in normal behavior

**Cleanup**:
```bash
git checkout main && git branch -D <created-branch>
```

---

### Test 5: Mixed SSH and HTTPS Remotes

**Objective**: Verify HTTPS remotes are still fetched when SSH agent unavailable.

**Steps**:
1. Add HTTPS remote if not present:
   ```bash
   git remote add https-test https://github.com/example/repo.git
   ```
2. Unset SSH_AUTH_SOCK:
   ```bash
   unset SSH_AUTH_SOCK
   ```
3. Run new-feature command:
   ```bash
   autospec new-feature "test mixed remotes"
   ```

**Expected Results**:
- [ ] SSH remotes are skipped
- [ ] HTTPS remotes are fetched (may fail auth but attempt is made)
- [ ] Command completes successfully
- [ ] Feature branch is created

**Cleanup**:
```bash
git checkout main && git branch -D <created-branch>
git remote remove https-test
```

---

### Test 6: Empty Repository (No Remotes)

**Objective**: Verify command works in repo with no remotes.

**Steps**:
1. Create a new empty repo:
   ```bash
   mkdir /tmp/empty-repo && cd /tmp/empty-repo
   git init
   git commit --allow-empty -m "Initial commit"
   ```
2. Run new-feature command:
   ```bash
   autospec new-feature "test empty repo"
   ```

**Expected Results**:
- [ ] Command completes successfully
- [ ] No fetch errors (nothing to fetch)
- [ ] Feature branch is created

**Cleanup**:
```bash
rm -rf /tmp/empty-repo
```

---

## Report Summaries

### Test Execution Date: ____________

### Tester: ____________

| Test # | Test Name | Status | Notes |
|--------|-----------|--------|-------|
| 1 | SSH Remote Skip | | |
| 2 | --no-fetch Flag | | |
| 3 | Timeout Behavior | | |
| 4 | Normal Operation | | |
| 5 | Mixed Remotes | | |
| 6 | Empty Repository | | |

### Issues Found:

(List any bugs, unexpected behaviors, or concerns discovered during testing)

### Overall Result: [ ] PASS / [ ] FAIL

### Additional Notes:

(Any other observations or recommendations)
