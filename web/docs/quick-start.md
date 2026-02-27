# Quick Start

## 1. Install the hook

In your git repository, run:

```bash
hourgit init
```

This installs a `post-checkout` git hook that silently records branch switches.

You can optionally assign the repo to a project:

```bash
hourgit init --project 'My Project'
```

## 2. Work normally

Time tracks automatically on every `git checkout`. Each branch switch creates a checkout entry that attributes time to the previous branch.

```bash
git checkout feature/auth
# ... work for 2 hours ...
git checkout fix/login-bug
# 2h attributed to feature/auth
```

## 3. Log non-git work

For meetings, reviews, research, or anything that isn't a branch switch:

```bash
hourgit log --duration 1h30m "standup"
hourgit log --from 9am --to 10:30am "code review"
```

## 4. View the interactive report

```bash
hourgit report
```

Navigate with arrow keys, press `e` to edit entries, `a` to add new ones. Checkout-derived time appears automatically (marked with `*`).

Press `s` to **submit** the period — this persists all generated entries and marks the period as complete.

## 5. Export a PDF

```bash
hourgit report --export pdf
```

This generates a PDF timesheet with a day-by-day breakdown grouped by task.
