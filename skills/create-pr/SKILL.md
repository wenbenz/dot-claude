---
name: create-pr
description: Create a new branch, apply changes in a git worktree (isolation), push without force, and open a pull request. Use when the user asks to "create a PR", "open a PR", "submit a pull request", "push to a new branch", or "create-pr".
allowed-tools: Read Write Bash(git branch*) Bash(git rev-parse*) Bash(git remote*) Bash(git worktree*) Bash(git -C * add*) Bash(git -C * commit*) Bash(git -C * push -u*) Bash(gh pr create*)
effort: medium
argument-hint: [branch-name]
---

# Create PR

Create a new branch, make changes in isolation using a git worktree, push the branch, and open a pull request.

```!
echo "Current branch: $(git branch --show-current 2>/dev/null || echo '(not a git repo)')"
echo "Repo root:      $(git rev-parse --show-toplevel 2>/dev/null || echo '(none)')"
echo "Remote origin:  $(git remote get-url origin 2>/dev/null || echo '(no origin)')"
```

## Steps

1. **Verify git repo** — if not inside a git repository, stop and tell the user.

2. **Determine branch name** — use `$ARGUMENTS` if provided; otherwise ask the user.

3. **Check for conflicts** — run `git branch --list <branch-name>`; if the branch already exists, ask the user before proceeding.

4. **Create a worktree** — creates an isolated checkout without touching the current working tree:
   ```
   git worktree add /tmp/<branch-name> -b <branch-name>
   ```

5. **Apply changes inside the worktree** — read source files from the repo root, then write or edit the corresponding files under `/tmp/<branch-name>/`. All modifications go into the worktree only.

6. **Commit inside the worktree**:
   ```
   git -C /tmp/<branch-name> add <files>
   git -C /tmp/<branch-name> commit -m "<message>"
   ```

7. **Push the branch** — never use `-f` or `--force`:
   ```
   git -C /tmp/<branch-name> push -u origin <branch-name>
   ```

8. **Open a pull request**:
   ```
   gh pr create --title "<title>" --body "<summary of changes>"
   ```

9. **Clean up the worktree**:
   ```
   git worktree remove /tmp/<branch-name>
   ```

10. **Confirm** — show the PR URL to the user.

## Rules

- Never use `git push --force` or `git push -f`
- All file edits must happen inside `/tmp/<branch-name>/`, never in the main checkout
- If `/tmp/<branch-name>` already exists, abort and tell the user
