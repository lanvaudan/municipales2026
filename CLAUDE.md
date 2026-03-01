# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Campaign website for "Unis pour Lanvaudan" — 2026 French municipal elections. Collects email subscribers (RGPD-compliant) and presents the candidate list and platform.

## Commands

```bash
# Run in development (default port 8024)
go run src/main.go

# Build production binary
bash bin/build.sh
# outputs bin/lanvaudan

# Run with custom port or debug mode
PORT=3000 go run src/main.go
GIN_MODE=debug go run src/main.go

# Verify database contents
bash bin/verifDB.sh

# Export subscriber emails (for mailing)
sqlite3 data/subscribers.db "SELECT email FROM subscribers;" > extraction_emails.csv

# Manage systemd service
bash bin/service.sh
```

Go requires CGO (for `go-sqlite3`): ensure `gcc` is available before building.

## Architecture

**Stack:** Go 1.25 + Gin (backend) · Vanilla HTML/CSS/JS + GSAP (frontend) · SQLite (data)

**Backend (`src/main.go`):** Single file (~110 lines). No ORM — raw SQL with parameterized queries. Three routes:
- `GET /` → serves `templates/index.html`
- `POST /subscribe` → validates email via `net.mail.ParseAddress`, inserts into SQLite, returns JSON; HTTP 409 on duplicate
- `GET /assets/*` → static file serving

**Database (`data/subscribers.db`):** Single table `subscribers(id, email UNIQUE, created_at)`. Auto-created on first run.

**Frontend (`templates/index.html`):** Monolithic file (~795 lines). CSS is embedded in `<style>` (intentional — reduces HTTP requests). JS is inline at the bottom. No build step.

## Frontend Conventions

- **CSS theming:** Use `:root` variables `--bg-color`, `--accent-color`, `--text-color` for all color values.
- **Scroll animations:** Add class `.reveal` to any block element to get automatic GSAP ScrollTrigger fade-in.
- **Candidate list:** The `#teamBody` table is generated client-side from the `candidates` JS array in the `<script>` block. Edit that array to update the team — do not duplicate HTML rows manually.
- **RGPD:** The `<p class="rgpd">` notice in the subscription form must never be hidden or removed.

## Roadmap (known TODOs)

- Rate-limiting on `POST /subscribe`
- Extract `<style>` to `/assets/style.css` when `index.html` becomes unwieldy
- CLI command to purge subscriber table
- Convert future image assets to `.webp`
