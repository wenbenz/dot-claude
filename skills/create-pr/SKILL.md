---
name: create-pr
description: Create a new branch, apply changes in a git worktree (isolation), push without force, and open a pull request. Use when the user asks to "create a PR", "open a PR", "submit a pull request", "push to a new branch", or "create-pr".
allowed-tools: Read Write Bash(git branch*) Bash(git rev-parse*) Bash(git remote*) Bash(git worktree*) Bash(git -C * add*) Bash(git -C * commit*) Bash(git -C * push -u*) Bash(gh pr create*)
effort: medium
argument-hint: [branch-name]
---

# Create PR

```!
git branch --show-current 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null && git remote get-url origin 2>/dev/null
```

1. **Verify git repo** — stop if not.
2. **Branch name** — use `$ARGUMENTS` if provided; else ask.
3. **Check conflicts** — `git branch --list <branch-name>`; if exists, ask before proceeding.
4. **Create worktree** for branch (follow worktree workflow rules for path).
5. **Apply changes** — read from repo root, write inside worktree.
6. **Commit** inside worktree.
7. **Push** (no `-f`) from worktree.
8. **Open PR**: `gh pr create --title "<title>" --body "<summary>"`
9. **Clean up** worktree after PR is open.
10. **Confirm** — show PR URL.

Never `git push --force` or `-f`.
