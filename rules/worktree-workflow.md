# Worktree Workflow

Before touching any file in a git repo:

1. **Find or create branch** — `git branch --list`; reuse existing related branch or create `feat/<topic>` / `fix/<topic>`.
2. **Check existing worktrees** — `git worktree list`; reuse if target branch already checked out.
3. **Work inside a worktree** — add if needed:
   ```
   git worktree add ./.worktrees/<branch-name> <branch-name>
   ```
   Edit files under `./.worktrees/<branch-name>/`, never main checkout. Use `git -C ./.worktrees/<branch-name>` for all git ops.
4. **Monitor PR** — poll with `gh pr view <number> --json state,reviewDecision,comments`; apply fixes in same worktree, commit, push. Keep worktree alive until merged or closed.
5. **Clean up** after merge:
   ```
   git worktree remove ./.worktrees/<branch-name>
   git worktree prune
   ```

Applies to all file edits, not just dev-pipeline runs.
