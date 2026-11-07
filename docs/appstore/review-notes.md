# App Review Notes

WeekTodo is a fully offline weekly task planner. No account or login is required.

## How to test

1. Launch the app. The current week is shown.
2. Add a task using the field at the bottom and press Enter or the add button.
3. Tap a task title to edit it. Tap the calendar icon to set a date. Tap the `…` button for subtasks, reorder, move, drop and delete.
4. Use the left and right arrows in the header to move between weeks.
5. Open the report button in the top-right to generate the weekly report.
6. Open Settings (gear icon) to change appearance, language, categories, WebDAV and data options.

## Optional features

- AI polish is optional. It requires the user to enter their own OpenAI-compatible endpoint and API key. It is not needed to review the core app.
- WebDAV sync is optional. It requires the user to enter their own WebDAV server. The local network permission prompt only appears when testing WebDAV on a LAN server.

## Data

All task data is stored locally in the app sandbox. The app contains no ads, analytics or tracking. Privacy details: https://github.com/codeskyblue/weektodo/blob/main/docs/privacy-policy.html

## Notes

- The app does not use in-app purchases.
- The app does not require push notifications, location, camera, photos or contacts.
- The update mechanism is disabled on iOS because App Store updates are handled by the system.
