# Worktree Workflow

Whenever making changes inside a git repository, follow these steps before touching any file:

1. **Find or create a branch** — run `git branch --list` to look for an existing branch related to this work. If one exists, use it. If not, create one with a descriptive name (e.g. `feat/<topic>` or `fix/<topic>`).

2. **Check existing worktrees** — run `git worktree list` to see what worktrees are already checked out. If one already tracks the target branch, reuse it instead of creating a new one.

3. **Make all changes inside a worktree** — add the branch as a worktree if needed:
   ```
   git worktree add ./.worktree/<branch-name> <branch-name>
   ```
   Write and edit files under `./.worktree/<branch-name>/`, never in the main checkout. Use `git -C ./.worktree/<branch-name>` for all git operations inside the worktree.

4. **Monitor the PR until merged** — after pushing, watch the PR for review comments or requested changes:
   - Use `gh pr view <number> --json state,reviewDecision,comments` to poll status.
   - If changes are requested, apply fixes inside the same worktree, commit, and push.
   - Keep the worktree alive until the PR is merged or closed.

5. **Clean up** — worktrees do not auto-delete; remove only after the PR is merged (or closed):
   ```
   git worktree remove ./.worktree/<branch-name>
   git worktree prune
   ```

This applies to all file edits, not just dev-pipeline runs.
