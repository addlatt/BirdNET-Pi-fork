#!/usr/bin/env bash
set -euo pipefail

# Ralph Loop for BirdNET-Pi-fork
# Runs Claude Code in a loop, one PRD story per iteration.
# Each iteration gets fresh context but persistent memory via git + progress.txt.

# Always run from repo root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

MAX_ITERATIONS="${1:-10}"
ITERATION=0

echo "=== Ralph Loop Starting ==="
echo "Max iterations: $MAX_ITERATIONS"
echo ""

while [ "$ITERATION" -lt "$MAX_ITERATIONS" ]; do
    ITERATION=$((ITERATION + 1))
    echo "=== Iteration $ITERATION of $MAX_ITERATIONS ==="

    # Check if all stories pass before starting
    REMAINING=$(jq '[.userStories[] | select(.passes == false)] | length' ralph/prd.json)
    if [ "$REMAINING" -eq 0 ]; then
        echo ""
        echo "=== All PRD stories complete! ==="
        echo "Finished in $((ITERATION - 1)) iterations."
        exit 0
    fi

    echo "Remaining stories: $REMAINING"
    echo ""

    # Run Claude Code with the ralph prompt
    claude --print --dangerously-skip-permissions -m claude-opus-4-6 "$(cat <<'PROMPT'
You are running inside a Ralph loop — an autonomous iteration loop.

## Your task

1. Read ralph/prd.json to find the FIRST user story where "passes": false
2. Read ralph/progress.txt for context from previous iterations
3. Implement that ONE story completely
4. Run verification: make test (for Go changes), and any other relevant checks
5. Update ralph/prd.json to set that story's "passes": true
6. Append a brief log entry to ralph/progress.txt with what you did and any learnings
7. Stage and commit all changes with a descriptive message

## Rules

- Only work on ONE story per iteration
- Run tests before marking a story as passing — if tests fail, fix them
- If you cannot complete a story, leave it as "passes": false and document why in ralph/progress.txt
- Keep commits atomic — one commit per story
- Do not modify stories you aren't working on
- Read CLAUDE.md for project conventions and architecture details

## Important paths

- PRD: ralph/prd.json
- Progress log: ralph/progress.txt
- Project guide: CLAUDE.md
- Go tests: make test
- Go lint: make lint
- Web build: make build-web
PROMPT
)"

    echo ""
    echo "=== Iteration $ITERATION complete ==="
    echo ""

    # Brief pause between iterations
    sleep 2
done

echo ""
echo "=== Ralph Loop finished (hit max iterations: $MAX_ITERATIONS) ==="
REMAINING=$(jq '[.userStories[] | select(.passes == false)] | length' ralph/prd.json)
echo "Remaining stories: $REMAINING"
