import { app } from './app.svelte';
import type { TaskRow } from './types';

/**
 * Splits pasted text into one task per non-empty line. All common newline
 * styles are supported, and surrounding whitespace is removed.
 */
export function parseTaskLines(text: string): string[] {
	return text
		.split(/\r\n|\r|\n/)
		.map((line) => line.trim())
		.filter((line) => line.length > 0);
}

/**
 * Creates a task when the empty space below the task list is clicked.
 * A click within one row below the last task continues that task's level;
 * clicking farther down starts a new top-level task.
 */
export function addTaskFromBlankArea(event: MouseEvent, weekId: string, rows: TaskRow[]): void {
	if (app.draggingTaskId) return;

	const last = rows.at(-1);
	if (!last) {
		app.addTask({ weekId });
		return;
	}

	// Keyboard and assistive-technology activation has no click coordinates.
	if (event.detail === 0) {
		app.addTask({ weekId });
		return;
	}

	const container = (event.currentTarget as HTMLElement).parentElement;
	const rowEl = container?.querySelector<HTMLElement>(
		`[data-task-id="${CSS.escape(last.task.id)}"]`
	);
	const rect = rowEl?.getBoundingClientRect();
	if (!rect || event.clientY - rect.bottom > rect.height) {
		app.addTask({ weekId });
		return;
	}

	// Keep the new task visible when a filter would hide it.
	const week = app.weekById(weekId);
	const day = last.task.day ?? (week && app.isCurrent(week) ? app.today : null);
	if (
		app.filter === 'done' ||
		app.filter === 'dropped' ||
		(app.filter === 'today' && day !== app.today)
	) {
		app.filter = 'all';
	}
	app.addSibling(last.task.id);
}
