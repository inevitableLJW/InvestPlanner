# InvestPlanner Agent Rules

## Database safety

- The local full stack must have exactly one confirmed canonical MySQL data volume. Never create or retain a second local database volume as a fallback, preview, test, or workaround.
- The exact canonical volume name and Compose project name belong in `docker-compose.yml`, not in this rules file.
- Start the local full stack with plain `docker compose up`; do not use `-p`, alternate Compose files, or storage overrides. Verify that Compose resolves MySQL to exactly one named volume.
- Treat the established Compose project name, MySQL volume name, mount type, mount source, and the main/dev mapping to that same volume as immutable safety configuration.
- Never modify those Compose database-storage settings, add a database override file, use `MYSQL_DATA_DIR` locally, or manually create a different database volume without the user's explicit approval for the exact proposed diff in the current conversation.
- Before every local startup, resolve both `docker-compose.yml` and `docker-compose.dev.yml` with `docker compose config`. They must resolve to the same project name and the same MySQL volume; otherwise stop without starting or changing anything.
- Never use `docker compose -p` or `COMPOSE_PROJECT_NAME` for this repository. Never start a Compose configuration that would create a second project or database volume.
- Before starting Docker services, report the resolved volume and verify it contains the expected application account and business data. Do not infer the correct database from row counts alone.
- Never create or switch to a temporary, preview, empty, or newly initialized database unless the user has explicitly approved that exact storage target and purpose in the current conversation.
- If more than one candidate database storage target exists, stop normal startup, identify them using read-only account and business-data evidence, obtain confirmation of the canonical target, and remove non-canonical targets only with explicit approval.
- If authentication to the existing database fails, stop and report the failure. Do not change the database volume, initialize another database, reset credentials, delete data, or substitute an empty database as a workaround.
- Never delete, reset, overwrite, or migrate away from the existing database volume without explicit approval for the exact operation and target.
- A successful startup must be verified against the expected database state and application account data; container health alone is insufficient.

## Git workflow

- Work directly in the current checkout using normal branches. Do not create or use Git worktrees.
- Before any Git write operation, report the current branch, `git status`, and the exact intended scope.
- Do not create branches, commit, amend, merge, rebase, reset, push, force-push, or archive OpenSpec changes without explicit approval for that operation.
