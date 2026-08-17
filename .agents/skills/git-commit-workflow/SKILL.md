---
name: git-commit-workflow
description: Rules for committing, bumping versions, and pushing to git. Use whenever asked to commit, save, bump version, push, push to remote, or release — or before you run any git write command yourself.
---

# ⚠️ CRITICAL: NEVER COMMIT WITHOUT EXPLICIT ORDER ⚠️

**STOP. READ THIS FIRST.**

- Do NOT run `git commit` unless the user says "commit"
- Do NOT run `git push` unless the user says "push"
- Do NOT run `git add -A` unless the user says "commit"
- Do NOT bump VERSION unless the user says "bump"

**ASK FIRST. WAIT FOR ORDER.**

---

# Git Commit / Bump / Push

1. `git pull` first, at the start of a session.
2. Never commit without an explicit order ("commit", "commit and push", "save"). Fixing a bug is not a commit order.
3. Never push without an explicit order. If told "commit and push", push right after committing — don't ask again. If only told "commit", stop there.
4. Commit messages start with `AGENT:`.
5. Before committing, show staged files + a short diff summary.
6. Commit and bump are independent — a commit order never implies a bump, and a bump order never implies a commit. Only bump when explicitly told to ("bump", "bump version", "release"), or if the repo's own convention is to bump on every commit (check first, don't assume). Bump BEFORE committing, never after. Always bump **patch** unless explicitly told to bump minor or major.
    - Node/Tauri: `package.json` (and `src-tauri/tauri.conf.json` if present, keeping both numbers equal).
    - Go: a version string, usually one of — a `VERSION` file, a `const Version = "x.y.z"` (e.g. in `version.go` or `main.go`). Check which one the repo actually uses before bumping; don't add a new mechanism.
7. Never commit secrets (.env, keys, tokens).
8. Confirm before destructive commands: force push, `reset --hard`, `checkout --` (discards changes), `clean -fd`, branch deletion.
9. Prior bumps/commits/pushes earlier in the conversation are not standing permission — each one needs its own explicit ask.
