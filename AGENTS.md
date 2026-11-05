# AGENTS

## Testing is mandatory

Every change must be verified before it is considered done.

Always test after every change. Do not report completion, commit, or release until the relevant checks have passed. If a check cannot be run, say so explicitly.

1. Backend changes: add or update Go unit tests and run `go test ./...`.
2. Frontend changes: run `npm run check` and `npm run build` in `frontend/`.
3. User-facing changes: verify the actual flow with Playwright (`playwright-cli`) against a temporary backend and the Vite dev server or built server.
   - Use a temporary `WEEKTODO_DATA_DIR` and non-default ports.
   - Create the required data through the API, then drive the UI.
   - Check the page result and the browser console for errors.
   - Stop the dev servers and delete temporary data/browser sessions afterwards.
4. Desktop changes: build the `.app` and verify the menu bar icon, window hide/show, and quit flow.
5. Fix regressions found by tests before reporting completion.

## Code quality

- Keep code small, direct and readable.
- Prefer typed models over ad-hoc maps.
- Reuse existing helpers, components and converters; do not duplicate logic across files.
- Keep parent/child state rules identical on the client and server.
- Do not add network dependencies for core task editing; everything must work offline.
- Avoid dead configuration and unused code.

## Data safety

- Never overwrite `data.json` non-atomically.
- Keep backward compatibility for older JSON without new fields.
- Do not silently drop user data on parse errors; recover from backups or fail loudly.
- Never return the AI API key from an HTTP endpoint or write it to logs.
