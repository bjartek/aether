# Layout Test

Test harness for table + inspector panel layout with syntax-highlighted code.

## Run

From the repo root:

```bash
go run cmd/layout/main.go
```

## What It Does

- **Left panel (30%)**: Table with dummy script/transaction names
- **Right panel (70%)**: Inspector showing `schedule_transaction.cdc` with:
  - Syntax highlighting (solarized-dark theme)
  - ANSI-aware wrapping at 160 characters (using reflow)
  - Scrollable viewport

## Controls

- `j` / `k` or `↓` / `↑`: Navigate table
- `q` or `Ctrl+C`: Quit

## Purpose

Tests the layout and code wrapping behavior we implemented for the runner view:
1. Split layout (table + inspector)
2. Code highlighting with chroma
3. ANSI-aware wrapping with reflow
4. Viewport scrolling

This lets you verify the 160-char wrapping looks good without running the full app.
