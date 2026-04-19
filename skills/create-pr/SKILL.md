---
name: create-pr
description: Create a new branch, apply changes in a git worktree (isolation), push without force, and open a pull request. Use when the user asks to "create a PR", "open a PR", "submit a pull request", "push to a new branch", or "create-pr".
allowed-tools: Read Write Bash(git branch*) Bash(git rev-parse*) Bash(git remote*) Bash(git worktree*) Bash(git -C * add*) Bash(git -C * commit*) Bash(git -C * push -u*) Bash(gh pr create*)
effort: medium
argument-hint: [branch-name]
---

# Create PR

Create branch, make changes in git worktree, push, open PR.

```!
git branch --show-current 2>/dev/null && git rev-parse --show-toplevel 2>/dev/null && git remote get-url origin 2>/dev/null
```

## Steps

1. **Verify git repo** — stop if not inside one.
2. **Branch name** — use `$ARGUMENTS` if provided; else ask.
3. **Check conflicts** — `git branch --list <branch-name>`; if exists, ask before proceeding.
4. **Create worktree**: `git worktree add /tmp/<branch-name> -b <branch-name>`
5. **Apply changes** — read from repo root, write under `/tmp/<branch-name>/`
6. **Commit**: `git -C /tmp/<branch-name> add <files> && git -C /tmp/<branch-name> commit -m "<message>"`
7. **Push** (no `-f`): `git -C /tmp/<branch-name> push -u origin <branch-name>`
8. **Open PR**: `gh pr create --title "<title>" --body "<summary>"`
9. **Clean up**: `git worktree remove /tmp/<branch-name>`
10. **Confirm** — show PR URL.

## Rules

- Never `git push --force` or `-f`
- All edits inside `/tmp/<branch-name>/` only
- If `/tmp/<branch-name>` exists, abort and tell user
