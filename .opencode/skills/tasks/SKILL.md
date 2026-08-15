---
name: tasks
description: How tasks work in this repo — the tasks/ folder, the task list in AGENTS.md, and the status markers agents must keep updated. Use whenever creating, picking up, or finishing a task.
---

# Tasks

## Where tasks live

1. One task per file, in `tasks/`, named `NNN-short-slug.md` (zero-padded, next free number).
2. Each file starts with a `Status:` line carrying one of the markers below, then the title and sections: Context, Requirements, Acceptance criteria, Progress (dated notes at the bottom).
3. Never delete a finished task file. Completed tasks stay in the folder, marked `[DONE]`.

## The task list in AGENTS.md

4. AGENTS.md has a `## Tasks` section listing every task file, one line per task.
5. Each line is `- NNN [MARKER] Title`. The marker goes right after the task number, at the start of the line.

## Status markers

6. `[TODO]` — not started (default for a new task).
7. `[IN_PROGRESS]` — an agent is actively working on it. Set this the moment you start.
8. `[DONE]` — complete, verified against the acceptance criteria.

## When to update

9. Creating a task → add the file in `tasks/` and its line in AGENTS.md, both with `[TODO]`.
10. Starting work on a task → set `[IN_PROGRESS]` in the AGENTS.md list **and** in the task file's `Status:` line, in the same change.
11. Finishing a task → verify the acceptance criteria, set `[DONE]` in both places, and add a short dated note to the task file's Progress section.
12. The task file's `Status:` line and its AGENTS.md line must always show the same marker — keep them in sync.
13. After a task is `[DONE]`, stop and report back to the user. Do not
    auto-continue into the next task in a loop — wait for an explicit order.
