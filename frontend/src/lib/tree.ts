import type { Filter, Stats, Task, TaskRow } from './types';

export function nowISO(): string {
	return new Date().toISOString();
}

export function newTask(text = '', day: string | null = null): Task {
	const task: Task = {
		id: 't_' + crypto.randomUUID().replaceAll('-', '').slice(0, 16),
		text,
		status: 'todo',
		children: [],
		createdAt: nowISO()
	};
	if (day) task.day = day;
	return task;
}

export function flatten(tasks: Task[], parent: Task | null = null, depth = 0): TaskRow[] {
	const rows: TaskRow[] = [];
	tasks.forEach((task, index) => {
		rows.push({ task, depth, parent, siblings: tasks, index, context: false });
		if (task.children.length > 0) {
			rows.push(...flatten(task.children, task, depth + 1));
		}
	});
	return rows;
}

export function visibleRows(tasks: Task[], filter: Filter, today: string): TaskRow[] {
	const matches = (task: Task): boolean => {
		switch (filter) {
			case 'today':
				return task.day === today;
			case 'todo':
				return task.status === 'todo';
			case 'done':
				return task.status === 'done';
			case 'dropped':
				return task.status === 'dropped';
			default:
				return true;
		}
	};
	const walk = (list: Task[], parent: Task | null, depth: number): TaskRow[] => {
		const rows: TaskRow[] = [];
		list.forEach((task, index) => {
			const childRows = walk(task.children, task, depth + 1);
			const matched = matches(task);
			if (!matched && childRows.length === 0) return;
			rows.push({
				task,
				depth,
				parent,
				siblings: list,
				index,
				context: !matched
			});
			rows.push(...childRows);
		});
		return rows;
	};
	return walk(tasks, null, 0);
}

export interface Location {
	task: Task;
	parent: Task | null;
	siblings: Task[];
	index: number;
	depth: number;
}

export function locate(tasks: Task[], taskId: string, parent: Task | null = null, depth = 0): Location | null {
	for (let index = 0; index < tasks.length; index++) {
		const task = tasks[index];
		if (task.id === taskId) {
			return { task, parent, siblings: tasks, index, depth };
		}
		const found = locate(task.children, taskId, task, depth + 1);
		if (found) return found;
	}
	return null;
}

export function findTask(tasks: Task[], taskId: string): Task | null {
	return locate(tasks, taskId)?.task ?? null;
}

export function countTodoDescendants(task: Task): number {
	let count = 0;
	const walk = (list: Task[]): void => {
		for (const child of list) {
			if (child.status === 'dropped') continue;
			if (child.status === 'todo') count++;
			walk(child.children);
		}
	};
	walk(task.children);
	return count;
}

export function isIndeterminate(task: Task): boolean {
	return task.status !== 'dropped' && task.status !== 'done' && countTodoDescendants(task) > 0;
}

export function completeSubtree(task: Task, at = nowISO()): void {
	if (task.status === 'dropped') return;
	task.status = 'done';
	task.completedAt ||= at;
	for (const child of task.children) {
		completeSubtree(child, at);
	}
}

export function uncompleteSubtree(task: Task): void {
	if (task.status === 'dropped') return;
	task.status = 'todo';
	task.completedAt = '';
	for (const child of task.children) {
		uncompleteSubtree(child);
	}
}

export function dropSubtree(task: Task, at = nowISO()): void {
	task.status = 'dropped';
	task.droppedAt ||= at;
	task.completedAt = '';
	for (const child of task.children) {
		dropSubtree(child, at);
	}
}

export function restoreSubtree(task: Task): void {
	task.status = 'todo';
	task.droppedAt = '';
	task.completedAt = '';
	for (const child of task.children) {
		restoreSubtree(child);
	}
}

export function recomputeParents(tasks: Task[], at = nowISO()): void {
	for (const task of tasks) {
		recomputeParents(task.children, at);
		if (task.status === 'dropped') continue;
		if (task.children.length === 0) {
			if (task.status === 'done' && !task.completedAt) task.completedAt = at;
			if (task.status !== 'done') task.completedAt = '';
			continue;
		}
		const active = task.children.filter((child) => child.status !== 'dropped');
		if (active.length === 0) {
			if (task.status !== 'done') task.completedAt = '';
			continue;
		}
		const allDone = active.every((child) => child.status === 'done');
		if (allDone) {
			task.status = 'done';
			const latest = active
				.map((child) => child.completedAt || '')
				.filter(Boolean)
				.sort((a, b) => Date.parse(a) - Date.parse(b))
				.at(-1);
			task.completedAt = latest || at;
		} else {
			task.status = 'todo';
			task.completedAt = '';
		}
	}
}

export function siblingInsert(tasks: Task[], taskId: string, task: Task): boolean {
	const found = locate(tasks, taskId);
	if (!found) return false;
	found.siblings.splice(found.index + 1, 0, task);
	return true;
}

export function appendChild(parent: Task, task: Task): void {
	parent.children.push(task);
}

export function removeTask(tasks: Task[], taskId: string): Task | null {
	const found = locate(tasks, taskId);
	if (!found) return null;
	found.siblings.splice(found.index, 1);
	return found.task;
}

export function indentTask(tasks: Task[], taskId: string): boolean {
	const found = locate(tasks, taskId);
	if (!found || found.index === 0) return false;
	if (found.depth + subtreeHeight(found.task) + 1 > 8) return false;
	const previous = found.siblings[found.index - 1];
	if (previous.status === 'dropped') return false;
	found.siblings.splice(found.index, 1);
	previous.children.push(found.task);
	return true;
}

export function subtreeHeight(task: Task): number {
	if (task.children.length === 0) return 1;
	return 1 + Math.max(...task.children.map(subtreeHeight));
}

export function outdentTask(tasks: Task[], taskId: string): boolean {
	const found = locate(tasks, taskId);
	if (!found || !found.parent) return false;
	const parentLocation = locate(tasks, found.parent.id);
	if (!parentLocation) return false;
	const parentChildren = found.parent.children;
	const childIndex = parentChildren.findIndex((child) => child.id === found.task.id);
	if (childIndex < 0) return false;
	parentChildren.splice(childIndex, 1);
	parentLocation.siblings.splice(parentLocation.index + 1, 0, found.task);
	return true;
}

export function moveSibling(tasks: Task[], taskId: string, delta: number): boolean {
	const found = locate(tasks, taskId);
	if (!found) return false;
	const target = found.index + delta;
	if (target < 0 || target >= found.siblings.length) return false;
	const [task] = found.siblings.splice(found.index, 1);
	found.siblings.splice(target, 0, task);
	return true;
}

export function leafStats(tasks: Task[]): Stats {
	const stats: Stats = { total: 0, done: 0, dropped: 0, unplanned: 0 };
	const walk = (list: Task[]): void => {
		for (const task of list) {
			if (task.children.length > 0) {
				walk(task.children);
				continue;
			}
			if (task.status === 'dropped') {
				stats.dropped++;
			} else if (task.status === 'done') {
				stats.total++;
				stats.done++;
			} else {
				stats.total++;
				if (!task.day) stats.unplanned++;
			}
		}
	};
	walk(tasks);
	return stats;
}

export function countNodes(task: Task): number {
	let count = 1;
	for (const child of task.children) count += countNodes(child);
	return count;
}
