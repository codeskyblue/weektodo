import { toast } from 'svelte-sonner';
import * as api from './api';
import {
	addDays,
	formatWeekTitle,
	nextWeekId,
	previousWeekId,
	weekIdFor,
	weekStart,
	todayISO
} from './date';
import type {
	CarryPreview,
	Category,
	ConfirmOptions,
	Filter,
	Meta,
	ReportMode,
	SettingsPublic,
	Stats,
	Task,
	TaskRow,
	Week
} from './types';
import {
	appendChild,
	completeSubtree,
	countNodes,
	countTodoDescendants,
	dropSubtree,
	findTask,
	indentTask,
	leafStats,
	locate,
	moveSibling,
	newTask,
	nowISO,
	outdentTask,
	recomputeParents,
	removeTask,
	restoreSubtree,
	siblingInsert,
	uncompleteSubtree,
	visibleRows
} from './tree';
import { i18n, t, type Language } from './i18n.svelte';
import { emitWails, onWails } from './wails';

export type SaveStatus = 'saved' | 'saving' | 'error';

export interface ConfirmState extends ConfirmOptions {
	open: boolean;
}

const emptyStats = (): Stats => ({ total: 0, done: 0, dropped: 0, unplanned: 0 });

function normalizeWeekSpan(value: number | undefined): 1 | 2 | 3 {
	return value === 2 || value === 3 ? value : 1;
}

function placeholderWeek(id: string): Week {
	const start = weekStart(id);
	return { id, start, end: addDays(start, 6), tasks: [] };
}

function messageOf(error: unknown): string {
	if (error instanceof api.ApiError) return error.message;
	if (error instanceof Error) return error.message;
	return String(error);
}

function taskLabel(task: Task): string {
	const text = task.text.trim();
	return text || t('（无标题任务）');
}

class AppState {
	categories = $state<Category[]>([]);
	activeCategoryId = $state('');
	categoryWeeks = $state<Record<string, string>>({});
	weeks = $state<Week[]>([]);
	anchorId = $state('');
	weekSpan = $state<1 | 2 | 3>(1);
	moveBusy = $state(false);
	draggingTaskId = $state('');
	dropTaskId = $state('');
	dropWeekId = $state('');
	meta = $state<Meta | null>(null);
	settings = $state<SettingsPublic | null>(null);
	currentWeekId = $state(weekIdFor(todayISO()));
	today = $state(todayISO());
	filter = $state<Filter>('all');
	loading = $state(true);
	navigating = $state(false);
	error = $state('');
	saveStatus = $state<SaveStatus>('saved');
	saveError = $state('');
	carryBusy = $state(false);
	settingsOpen = $state(false);
	settingsSection = $state('ai');
	pendingUpdateCheck = $state(false);
	reportOpen = $state(false);
	reportMode = $state<ReportMode>('brief');
	reportMarkdown = $state('');
	reportStats = $state<Stats>(emptyStats());
	reportLoading = $state(false);
	reportError = $state('');
	aiMarkdown = $state('');
	aiLoading = $state(false);
	confirm = $state<ConfirmState>({
		open: false,
		title: '',
		description: '',
		confirmLabel: t('确定')
	});
	quickAddFocus = $state(0);
	focusRequest = $state<{ id: string; caret: 'start' | 'end'; nonce: number } | null>(null);

	private dirtyWeeks = new Set<string>();
	private saving = false;
	private revision = 0;
	private saveTimer: ReturnType<typeof setTimeout> | null = null;
	private clock: ReturnType<typeof setInterval> | null = null;
	private focusSeq = 0;
	private loadSeq = 0;
	private wailsBound = false;
	private autoSyncTimer: ReturnType<typeof setTimeout> | null = null;
	private autoSyncInterval: ReturnType<typeof setInterval> | null = null;
	private autoSyncInFlight = false;
	private autoSyncQueued = false;
	private autoSyncErrorShown = false;
	private categoryStateTimer: ReturnType<typeof setTimeout> | null = null;

	get activeCategory(): Category | null {
		return this.categories.find((category) => category.id === this.activeCategoryId) ?? null;
	}

	get activeCategories(): Category[] {
		return this.categories.filter((category) => !category.archived);
	}

	get week(): Week | null {
		return this.weeks.find((week) => week.id === this.anchorId) ?? null;
	}

	get rows(): TaskRow[] {
		return this.week ? this.rowsFor(this.week) : [];
	}

	rowsFor(week: Week): TaskRow[] {
		return visibleRows(week.tasks, this.filter, this.today);
	}

	get stats(): Stats {
		return this.week ? leafStats(this.week.tasks) : emptyStats();
	}

	statsFor(week: Week): Stats {
		return leafStats(week.tasks);
	}

	get progress(): number {
		return this.week ? this.progressFor(this.week) : 0;
	}

	progressFor(week: Week): number {
		const stats = this.statsFor(week);
		return stats.total === 0 ? 0 : Math.round((stats.done / stats.total) * 100);
	}

	weekById(id: string): Week | null {
		return this.weeks.find((week) => week.id === id) ?? null;
	}

	weekOfTask(taskId: string): Week | null {
		for (const week of this.weeks) {
			if (findTask(week.tasks, taskId)) return week;
		}
		return null;
	}

	isCurrent(week: Week): boolean {
		return week.id === this.currentWeekId;
	}

	get reportText(): string {
		return this.aiMarkdown || this.reportMarkdown;
	}

	get isCurrentWeek(): boolean {
		return this.week?.id === this.currentWeekId;
	}

	async init(): Promise<void> {
		this.loading = true;
		this.error = '';
		if (!this.wailsBound) {
			this.wailsBound = true;
			onWails('weektodo:focus-quick-add', () => {
				this.quickAddFocus++;
			});
			onWails('weektodo:check-updates', () => {
				this.openSettings('update');
				this.pendingUpdateCheck = true;
			});
			onWails('weektodo:window-hiding', () => {
				void this.flushSave();
			});
		}
		try {
			const [meta, settings, categoryResult] = await Promise.all([
				api.getMeta(),
				api.getSettings(),
				api.listCategories()
			]);
			this.meta = meta;
			this.settings = settings;
			this.categories = categoryResult.categories;
			this.categoryWeeks = settings.categoryWeeks ?? {};
			this.weekSpan = normalizeWeekSpan(settings.weekSpan);
			const language: Language =
				settings.language === 'zh' || settings.language === 'en' ? settings.language : i18n.language;
			i18n.setLanguage(language);
			if (!settings.language) void api.saveLanguage(language);
			emitWails('weektodo:changed');
			this.currentWeekId = meta.currentWeekId;
			const params = new URLSearchParams(window.location.search);
			const category = this.resolveCategory(params.get('category') ?? settings.activeCategory ?? '');
			const wanted = params.get('week');
			const initial =
				wanted && /^\d{4}-W\d{2}$/.test(wanted)
					? wanted
					: (this.categoryWeeks[category] ?? meta.currentWeekId);
			await this.loadWindow(category, initial);
			this.startClock();
		} catch (error) {
			this.error = messageOf(error);
		} finally {
			this.loading = false;
		}
		this.refreshAutoSync();
		if (this.settings?.webdav.autoSync) this.scheduleAutoSync(1500);
	}

	private startClock(): void {
		if (this.clock) clearInterval(this.clock);
		this.clock = setInterval(() => {
			const next = todayISO();
			if (next === this.today) return;
			this.today = next;
			this.currentWeekId = weekIdFor(next);
		}, 30_000);
	}

	private resolveCategory(preferred: string): string {
		const available = this.categories.filter((category) => !category.archived);
		if (preferred && available.some((category) => category.id === preferred)) {
			return preferred;
		}
		const fallback = this.settings?.activeCategory ?? '';
		if (fallback && available.some((category) => category.id === fallback)) {
			return fallback;
		}
		return available[0]?.id ?? this.categories[0]?.id ?? 'default';
	}

	private async refreshWindow(): Promise<void> {
		const anchorId = this.anchorId;
		const category = this.activeCategoryId;
		if (!anchorId || !category) return;
		const token = ++this.loadSeq;
		try {
			const weeks = await Promise.all(
				this.windowIds(anchorId).map(async (id) => {
					return (await api.getWeekOrNull(category, id)) ?? placeholderWeek(id);
				})
			);
			if (token !== this.loadSeq) return;
			this.weeks = weeks;
		} catch {
			// A background refresh should never disturb the current view.
		}
	}

	async loadWindow(categoryId: string, anchorId: string): Promise<void> {
		this.endTaskDrag();
		const token = ++this.loadSeq;
		this.navigating = true;
		this.error = '';
		await this.flushSave();
		await this.waitForSave();
		try {
			// The anchor week is created on demand, while neighbouring weeks
			// stay in memory until something is actually written to them.
			const weeks = await Promise.all(
				this.windowIds(anchorId).map(async (id) => {
					if (id === anchorId) return (await api.createWeek(categoryId, id)).week;
					return (await api.getWeekOrNull(categoryId, id)) ?? placeholderWeek(id);
				})
			);
			if (token !== this.loadSeq) return;
			this.activeCategoryId = categoryId;
			this.weeks = weeks;
			this.anchorId = anchorId;
			this.categoryWeeks = { ...this.categoryWeeks, [categoryId]: anchorId };
			this.persistCategoryState();
			this.currentWeekId = this.currentWeekId || weekIdFor(todayISO());
			this.filter = 'all';
			this.reportMarkdown = '';
			this.aiMarkdown = '';
			this.reportError = '';
			this.saveStatus = 'saved';
			this.dirtyWeeks.clear();
			this.revision++;
			const url = new URL(window.location.href);
			url.searchParams.set('category', categoryId);
			url.searchParams.set('week', anchorId);
			window.history.replaceState(null, '', url);
		} catch (error) {
			if (token === this.loadSeq) this.error = messageOf(error);
		} finally {
			if (token === this.loadSeq) this.navigating = false;
		}
	}

	private persistCategoryState(): void {
		if (!this.activeCategoryId) return;
		if (this.categoryStateTimer) clearTimeout(this.categoryStateTimer);
		this.categoryStateTimer = setTimeout(() => {
			this.categoryStateTimer = null;
			void api
				.saveCategoryState(this.activeCategoryId, this.categoryWeeks)
				.then((settings) => {
					this.settings = settings;
				})
				.catch(() => {
					// A failed preference save must not disturb editing.
				});
		}, 500);
	}

	private windowIds(anchorId: string): string[] {
		switch (this.weekSpan) {
			case 3:
				return [previousWeekId(anchorId), anchorId, nextWeekId(anchorId)];
			case 2:
				return [previousWeekId(anchorId), anchorId];
			default:
				return [anchorId];
		}
	}

	async goToWeek(id: string): Promise<void> {
		if (id === this.anchorId) return;
		await this.loadWindow(this.activeCategoryId, id);
	}

	async switchCategory(id: string): Promise<void> {
		if (!id || id === this.activeCategoryId) return;
		const category = this.categories.find((item) => item.id === id);
		if (!category || category.archived) return;
		await this.flushSave();
		await this.waitForSave();
		const week = this.categoryWeeks[id] ?? this.currentWeekId;
		await this.loadWindow(id, week);
	}

	async goToday(): Promise<void> {
		await this.goToWeek(this.currentWeekId);
	}

	async goRelative(delta: number): Promise<void> {
		if (!this.week) return;
		await this.goToWeek(weekIdFor(addDays(this.week.start, delta * 7)));
	}

	requestFocus(id: string, caret: 'start' | 'end' = 'end'): void {
		this.focusRequest = { id, caret, nonce: ++this.focusSeq };
	}

	markDirty(weekId = this.anchorId): void {
		if (!weekId) return;
		this.dirtyWeeks.add(weekId);
		this.revision++;
		this.saveStatus = 'saving';
		if (this.saveTimer) clearTimeout(this.saveTimer);
		this.saveTimer = setTimeout(() => {
			this.saveTimer = null;
			void this.flushSave();
		}, 350);
	}

	async flushSave(): Promise<void> {
		if (this.saveTimer) {
			clearTimeout(this.saveTimer);
			this.saveTimer = null;
		}
		if (this.dirtyWeeks.size === 0 || this.saving) return;
		const ids = [...this.dirtyWeeks];
		this.dirtyWeeks.clear();
		this.saving = true;
		this.saveStatus = 'saving';
		const revision = this.revision;
		let failure: unknown = null;
		for (const id of ids) {
			const week = this.weekById(id);
			if (!week) continue;
			const tasks = JSON.parse(JSON.stringify(week.tasks)) as Task[];
			try {
				await api.saveWeek(this.activeCategoryId, id, tasks);
			} catch (error) {
				failure ??= error;
				this.dirtyWeeks.add(id);
			}
		}
		this.saving = false;
		if (failure) {
			this.saveStatus = 'error';
			this.saveError = messageOf(failure);
			toast.error(t('保存失败'), {
				description: this.saveError,
				action: {
					label: t('重试'),
					onClick: () => this.retrySave()
				}
			});
			return;
		}
		if (revision === this.revision) {
			this.saveStatus = 'saved';
			this.saveError = '';
		}
		emitWails('weektodo:changed');
		this.scheduleAutoSync();
		if (this.dirtyWeeks.size > 0) void this.flushSave();
	}

	retrySave(): void {
		void this.flushSave();
	}

	private async waitForSave(): Promise<void> {
		for (let attempt = 0; attempt < 120 && this.saving; attempt++) {
			await new Promise((resolve) => setTimeout(resolve, 50));
		}
	}

	private replaceWeek(week: Week): void {
		const index = this.weeks.findIndex((item) => item.id === week.id);
		if (index >= 0) this.weeks[index] = week;
		else this.weeks.push(week);
		this.weeks = [...this.weeks].sort((a, b) => a.start.localeCompare(b.start));
	}

	addTask(
		options: { text?: string; parentId?: string; day?: string | null; weekId?: string } = {}
	): Task | null {
		const week = this.weekById(options.weekId ?? this.anchorId);
		if (!week) return null;
		const day =
			options.day !== undefined
				? options.day
				: this.isCurrent(week)
					? this.today
					: null;
		const task = newTask(options.text ?? '', day);
		if (options.parentId) {
			const parent = findTask(week.tasks, options.parentId);
			if (!parent || parent.status === 'dropped') return null;
			appendChild(parent, task);
		} else {
			week.tasks.push(task);
		}
		recomputeParents(week.tasks);
		if (this.filter === 'done' || this.filter === 'dropped') this.filter = 'all';
		if (this.filter === 'today' && day !== this.today) this.filter = 'all';
		this.requestFocus(task.id, 'end');
		this.markDirty(week.id);
		return task;
	}

	addSibling(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found) return;
		const task = newTask('', found.task.day ?? (this.isCurrent(week) ? this.today : null));
		siblingInsert(week.tasks, taskId, task);
		recomputeParents(week.tasks);
		this.requestFocus(task.id, 'end');
		this.markDirty(week.id);
	}

	addChild(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found || found.task.status === 'dropped') return;
		const task = newTask('', found.task.day ?? (this.isCurrent(week) ? this.today : null));
		appendChild(found.task, task);
		recomputeParents(week.tasks);
		this.requestFocus(task.id, 'end');
		this.markDirty(week.id);
	}

	updateText(taskId: string, text: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const task = findTask(week.tasks, taskId);
		if (!task) return;
		task.text = text;
		task.updatedAt = nowISO();
		this.markDirty(week.id);
	}

	setDay(taskId: string, day: string | null): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const task = findTask(week.tasks, taskId);
		if (!task) return;
		if (day && (day < week.start || day > week.end)) return;
		if (day) task.day = day;
		else delete task.day;
		task.updatedAt = nowISO();
		this.markDirty(week.id);
	}

	toggleTask(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found) return;
		const task = found.task;
		if (task.status === 'dropped') return;
		const pending = countTodoDescendants(task);
		if (task.children.length > 0 && pending > 0) {
			this.askConfirm({
				title: t('完成父任务？'),
				description: t('「{title}」还有 {count} 个未完成的子任务。完成后会把这些子任务一起勾选。', { title: taskLabel(task), count: pending }),
				confirmLabel: t('全部完成'),
				onConfirm: () => {
					completeSubtree(task);
					recomputeParents(week.tasks);
					this.markDirty(week.id);
				}
			});
			return;
		}
		if (task.status === 'done') uncompleteSubtree(task);
		else completeSubtree(task);
		recomputeParents(week.tasks);
		this.markDirty(week.id);
	}

	dropTask(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found || found.task.status === 'dropped') return;
		const task = found.task;
		const total = countNodes(task);
		const apply = (): void => {
			dropSubtree(task);
			recomputeParents(week.tasks);
			this.markDirty(week.id);
			toast.success(t('已废弃'), {
				action: {
					label: t('撤销'),
					onClick: () => this.restoreTask(taskId)
				}
			});
		};
		if (total > 1) {
			this.askConfirm({
				title: t('废弃任务？'),
				description: t('「{title}」及其 {count} 个子任务都会被废弃，且不计入完成率和周报。', { title: taskLabel(task), count: total - 1 }),
				confirmLabel: t('废弃'),
				destructive: true,
				onConfirm: apply
			});
			return;
		}
		apply();
	}

	restoreTask(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found) return;
		restoreSubtree(found.task);
		recomputeParents(week.tasks);
		this.markDirty(week.id);
	}

	deleteTask(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found) return;
		const task = found.task;
		const total = countNodes(task);
		const rows = this.rowsFor(week);
		const rowIndex = rows.findIndex((row) => row.task.id === taskId);
		const fallbackFocus = rowIndex > 0 ? rows[rowIndex - 1].task.id : null;
		this.askConfirm({
			title: t('删除任务？'),
			description:
				total > 1
					? t('「{title}」及其 {count} 个子任务会被永久删除。', { title: taskLabel(task), count: total - 1 })
					: t('「{title}」会被永久删除。', { title: taskLabel(task) }),
			confirmLabel: t('删除'),
			destructive: true,
			onConfirm: () => {
				removeTask(week.tasks, taskId);
				recomputeParents(week.tasks);
				if (fallbackFocus) this.requestFocus(fallbackFocus, 'end');
				this.markDirty(week.id);
			}
		});
	}

	removeEmptyTask(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const found = locate(week.tasks, taskId);
		if (!found) return;
		if (found.task.text.trim() !== '' || found.task.children.length > 0) return;
		const rows = this.rowsFor(week);
		const rowIndex = rows.findIndex((row) => row.task.id === taskId);
		const fallbackFocus = rowIndex > 0 ? rows[rowIndex - 1].task.id : null;
		removeTask(week.tasks, taskId);
		recomputeParents(week.tasks);
		if (fallbackFocus) this.requestFocus(fallbackFocus, 'end');
		this.markDirty(week.id);
	}

	indent(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		const task = findTask(week.tasks, taskId);
		if (!task) return;
		if (!indentTask(week.tasks, taskId)) return;
		task.updatedAt = nowISO();
		recomputeParents(week.tasks);
		this.requestFocus(taskId, 'start');
		this.markDirty(week.id);
	}

	outdent(taskId: string): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		if (!outdentTask(week.tasks, taskId)) return;
		const task = findTask(week.tasks, taskId);
		if (task) task.updatedAt = nowISO();
		recomputeParents(week.tasks);
		this.requestFocus(taskId, 'start');
		this.markDirty(week.id);
	}

	move(taskId: string, delta: number): void {
		const week = this.weekOfTask(taskId);
		if (!week) return;
		if (!moveSibling(week.tasks, taskId, delta)) return;
		const task = findTask(week.tasks, taskId);
		if (task) task.updatedAt = nowISO();
		this.requestFocus(taskId, 'start');
		this.markDirty(week.id);
	}

	askConfirm(options: ConfirmOptions): void {
		this.confirm = {
			open: true,
			title: options.title,
			description: options.description ?? '',
			confirmLabel: options.confirmLabel ?? t('确定'),
			destructive: options.destructive ?? false,
			onConfirm: options.onConfirm
		};
	}

	async confirmAction(): Promise<void> {
		const action = this.confirm.onConfirm;
		this.confirm.open = false;
		await action?.();
	}

	cancelConfirm(): void {
		this.confirm.open = false;
	}

	async openCarryOver(): Promise<void> {
		if (!this.week || this.carryBusy) return;
		this.carryBusy = true;
		try {
			const preview: CarryPreview = await api.carryPreview(this.activeCategoryId, this.week.id);
			if (preview.reason === 'no_previous_week') {
				toast.info(t('上周还没有记录'), { description: t('没有可以带入的任务。') });
				return;
			}
			if (preview.count === 0) {
				toast.info(
					preview.skipped > 0 ? t('上周未完成的任务已经全部带入本周') : t('上周没有未完成的任务')
				);
				return;
			}
			const previewText = preview.tasks
				.slice(0, 6)
				.map((task) => `${'    '.repeat(task.depth)}· ${task.text || t('（无标题任务）')}`)
				.join('\n');
			const more = preview.tasks.length > 6 ? '\n' + t('… 还有 {count} 个', { count: preview.tasks.length - 6 }) : '';
			this.askConfirm({
				title: t('带入上周未完成任务'),
				description: t('将带入 {count} 个任务{skipped}：\n\n{preview}{more}', { count: preview.count, skipped: preview.skipped > 0 ? t('，跳过 {count} 个已带入的任务', { count: preview.skipped }) : '', preview: previewText, more }),
				confirmLabel: t('带入'),
				onConfirm: async () => {
					const weekId = this.week?.id;
					if (!weekId) return;
					this.carryBusy = true;
					try {
						const result = await api.carryOver(this.activeCategoryId, weekId);
						this.replaceWeek(result.week);
						this.dirtyWeeks.delete(result.week.id);
						this.reportMarkdown = '';
						this.aiMarkdown = '';
						this.saveStatus = 'saved';
						this.revision++;
						if (result.created > 0) {
							toast.success(t('已带入 {count} 个任务', { count: result.created }), {
								description: result.skipped > 0 ? t('跳过 {count} 个已带入的任务', { count: result.skipped }) : undefined
							});
						} else {
							toast.info(t('没有新的任务可带入'));
						}
						emitWails('weektodo:changed');
					} catch (error) {
						toast.error(t('带入失败'), { description: messageOf(error) });
					} finally {
						this.carryBusy = false;
					}
				}
			});
		} catch (error) {
			toast.error(t('无法读取上周任务'), { description: messageOf(error) });
		} finally {
			this.carryBusy = false;
		}
	}

	async openReport(): Promise<void> {
		this.reportOpen = true;
		await this.refreshReport();
	}

	async setReportMode(mode: ReportMode): Promise<void> {
		if (mode === this.reportMode) return;
		this.reportMode = mode;
		this.aiMarkdown = '';
		await this.refreshReport();
	}

	async refreshReport(): Promise<void> {
		if (!this.week) return;
		this.reportLoading = true;
		this.reportError = '';
		try {
			const response = await api.getReport(this.activeCategoryId, this.week.id, this.reportMode);
			this.reportMarkdown = response.markdown;
			this.reportStats = response.stats;
		} catch (error) {
			this.reportError = messageOf(error);
		} finally {
			this.reportLoading = false;
		}
	}

	async polishReport(): Promise<void> {
		if (!this.week) return;
		if (!this.settings?.ai.configured) {
			this.openSettings('ai');
			toast.info(t('请先配置 AI 接口'));
			return;
		}
		this.aiLoading = true;
		try {
			const response = await api.polishReport(this.activeCategoryId, this.week.id, this.reportMode);
			this.aiMarkdown = response.markdown;
			toast.success(t('AI 周报已生成'));
		} catch (error) {
			toast.error(t('AI 润色失败'), { description: messageOf(error) });
		} finally {
			this.aiLoading = false;
		}
	}

	clearAIReport(): void {
		this.aiMarkdown = '';
	}

	async setLanguage(language: Language): Promise<void> {
		i18n.setLanguage(language);
		this.aiMarkdown = '';
		if (this.reportOpen) void this.refreshReport();
		try {
			this.settings = await api.saveLanguage(language);
			emitWails('weektodo:changed');
		} catch (error) {
			toast.error(t('保存设置失败'), { description: messageOf(error) });
		}
	}

	async saveSettings(draft: api.SettingsDraft): Promise<boolean> {
		try {
			this.settings = await api.saveSettings(draft);
			this.autoSyncErrorShown = false;
			this.refreshAutoSync();
			if (this.settings?.webdav.autoSync) this.scheduleAutoSync(1500);
			toast.success(t('设置已保存'));
			return true;
		} catch (error) {
			toast.error(t('保存设置失败'), { description: messageOf(error) });
			return false;
		}
	}

	async testAISettings(draft: { baseUrl: string; model: string; apiKey: string }): Promise<boolean> {
		try {
			await api.testAI(draft);
			toast.success(t('AI 连接成功'));
			return true;
		} catch (error) {
			toast.error(t('AI 连接失败'), { description: messageOf(error) });
			return false;
		}
	}

	async testWebDAVSettings(draft: api.WebDAVDraft): Promise<boolean> {
		try {
			await api.testWebDAV(draft);
			toast.success(t('WebDAV 连接成功'));
			return true;
		} catch (error) {
			toast.error(t('WebDAV 连接失败'), { description: messageOf(error) });
			return false;
		}
	}

	async syncWebDAV(draft: api.WebDAVDraft): Promise<api.WebDAVSyncResult | null> {
		const weekID = this.anchorId;
		await this.flushSave();
		await this.waitForSave();
		await this.flushSave();
		try {
			const result = await api.syncWebDAV(draft);
			this.autoSyncErrorShown = false;
			toast.success(
				t('同步完成：{weeks} 周 / {tasks} 个任务', {
					weeks: result.weeks,
					tasks: result.tasks
				})
			);
			await this.refreshCategories();
			if (weekID) {
				await this.refreshWindow();
			}
			return result;
		} catch (error) {
			toast.error(t('同步失败'), { description: messageOf(error) });
			return null;
		}
	}

	async refreshCategories(): Promise<void> {
		const result = await api.listCategories();
		this.categories = result.categories;
		const current = this.categories.find(
			(category) => category.id === this.activeCategoryId && !category.archived
		);
		if (current) return;
		const next = this.resolveCategory(this.settings?.activeCategory ?? '');
		if (next) {
			await this.loadWindow(next, this.categoryWeeks[next] ?? this.currentWeekId);
		}
	}

	async saveCategory(
		id: string | null,
		draft: { name: string; color: string; archived?: boolean; order?: number }
	): Promise<boolean> {
		try {
			if (id) await api.updateCategory(id, draft);
			else await api.createCategory(draft.name, draft.color);
			await this.refreshCategories();
			return true;
		} catch (error) {
			toast.error(t('保存分类失败'), { description: messageOf(error) });
			return false;
		}
	}

	private refreshAutoSync(): void {
		if (!this.settings?.webdav.autoSync) {
			if (this.autoSyncInterval) {
				clearInterval(this.autoSyncInterval);
				this.autoSyncInterval = null;
			}
			if (this.autoSyncTimer) {
				clearTimeout(this.autoSyncTimer);
				this.autoSyncTimer = null;
			}
			return;
		}
		if (!this.autoSyncInterval) {
			this.autoSyncInterval = setInterval(() => void this.autoSync(), 5 * 60_000);
		}
	}

	private scheduleAutoSync(delay = 8000): void {
		if (!this.settings?.webdav.autoSync) return;
		if (this.autoSyncTimer) clearTimeout(this.autoSyncTimer);
		this.autoSyncTimer = setTimeout(() => {
			this.autoSyncTimer = null;
			void this.autoSync();
		}, delay);
	}

	private async autoSync(): Promise<void> {
		if (!this.settings?.webdav.autoSync || !this.settings.webdav.url) return;
		if (this.autoSyncTimer) {
			clearTimeout(this.autoSyncTimer);
			this.autoSyncTimer = null;
		}
		if (this.autoSyncInFlight) {
			this.autoSyncQueued = true;
			return;
		}
		if (this.loading || this.navigating || this.reportOpen || this.confirm.open) return;
		await this.flushSave();
		await this.waitForSave();
		if (this.saving || this.dirtyWeeks.size > 0) {
			this.autoSyncQueued = true;
			return;
		}
		const revision = this.revision;
		this.autoSyncInFlight = true;
		try {
			await api.syncWebDAV({
				url: this.settings.webdav.url,
				username: this.settings.webdav.username,
				password: '',
				remoteDir: this.settings.webdav.remoteDir
			});
			this.autoSyncErrorShown = false;
			if (revision === this.revision) {
				await this.refreshCategories();
				await this.refreshWindow();
			} else {
				this.autoSyncQueued = true;
			}
		} catch (error) {
			if (!this.autoSyncErrorShown) {
				this.autoSyncErrorShown = true;
				toast.error(t('自动同步失败'), { description: messageOf(error) });
			}
		} finally {
			this.autoSyncInFlight = false;
			if (this.autoSyncQueued) {
				this.autoSyncQueued = false;
				this.scheduleAutoSync(2000);
			}
		}
	}

	async setWeekSpan(span: 1 | 2 | 3): Promise<void> {
		if (span === this.weekSpan) return;
		await this.flushSave();
		await this.waitForSave();
		this.weekSpan = span;
		try {
			this.settings = await api.saveWeekSpan(span);
		} catch (error) {
			toast.error(t('保存设置失败'), { description: messageOf(error) });
		}
		if (this.activeCategoryId && this.anchorId) {
			await this.loadWindow(this.activeCategoryId, this.anchorId);
		}
	}

	moveOptions(taskId: string): { week: Week; direction: 'previous' | 'next' }[] {
		const week = this.weekOfTask(taskId);
		if (!week || this.weeks.length < 2) return [];
		const index = this.weeks.findIndex((item) => item.id === week.id);
		if (index < 0) return [];
		const options: { week: Week; direction: 'previous' | 'next' }[] = [];
		if (index > 0) options.push({ week: this.weeks[index - 1], direction: 'previous' });
		if (index < this.weeks.length - 1) {
			options.push({ week: this.weeks[index + 1], direction: 'next' });
		}
		return options;
	}

	async moveTaskToWeek(taskId: string, toWeekId: string, parentTaskId = ''): Promise<void> {
		if (this.moveBusy) return;
		const from = this.weekOfTask(taskId);
		if (!from) return;
		if (from.id === toWeekId && !parentTaskId) return;
		this.moveBusy = true;
		await this.flushSave();
		await this.waitForSave();
		await this.flushSave();
		try {
			const result = await api.moveTask(
				this.activeCategoryId,
				from.id,
				taskId,
				toWeekId,
				parentTaskId
			);
			this.replaceWeek(result.from);
			this.replaceWeek(result.to);
			this.dirtyWeeks.delete(result.from.id);
			this.dirtyWeeks.delete(result.to.id);
			toast.success(t('已移到 {week}', { week: formatWeekTitle(result.to.id) }));
			emitWails('weektodo:changed');
		} catch (error) {
			toast.error(t('移动失败'), { description: messageOf(error) });
		} finally {
			this.moveBusy = false;
		}
	}

	startTaskDrag(taskId: string): void {
		this.draggingTaskId = taskId;
		this.dropTaskId = '';
		this.dropWeekId = '';
	}

	endTaskDrag(): void {
		this.draggingTaskId = '';
		this.dropTaskId = '';
		this.dropWeekId = '';
	}

	canDropOnTask(targetTaskId: string): boolean {
		const dragId = this.draggingTaskId;
		if (!dragId || dragId === targetTaskId) return false;
		const sourceWeek = this.weekOfTask(dragId);
		const targetWeek = this.weekOfTask(targetTaskId);
		if (!sourceWeek || !targetWeek) return false;
		const source = findTask(sourceWeek.tasks, dragId);
		const target = findTask(targetWeek.tasks, targetTaskId);
		if (!source || !target || target.status === 'dropped') return false;
		return findTask(source.children, targetTaskId) === null;
	}

	setTaskDropTarget(targetTaskId: string): void {
		if (!this.canDropOnTask(targetTaskId)) return;
		this.dropTaskId = targetTaskId;
		this.dropWeekId = '';
	}

	clearTaskDropTarget(targetTaskId: string): void {
		if (this.dropTaskId === targetTaskId) this.dropTaskId = '';
	}

	setWeekDropTarget(weekId: string): void {
		if (!this.draggingTaskId || this.dropTaskId) return;
		this.dropWeekId = weekId;
	}

	clearWeekDropTarget(weekId: string): void {
		if (this.dropWeekId === weekId) this.dropWeekId = '';
	}

	async dropOnTask(targetTaskId: string): Promise<void> {
		const dragId = this.draggingTaskId;
		if (!dragId || !this.canDropOnTask(targetTaskId)) {
			this.endTaskDrag();
			return;
		}
		const sourceWeek = this.weekOfTask(dragId);
		const targetWeek = this.weekOfTask(targetTaskId);
		this.endTaskDrag();
		if (!sourceWeek || !targetWeek) return;
		if (sourceWeek.id !== targetWeek.id) {
			await this.moveTaskToWeek(dragId, targetWeek.id, targetTaskId);
			return;
		}
		const task = removeTask(sourceWeek.tasks, dragId);
		const parent = findTask(sourceWeek.tasks, targetTaskId);
		if (!task || !parent) return;
		parent.children.push(task);
		task.updatedAt = nowISO();
		recomputeParents(sourceWeek.tasks);
		this.markDirty(sourceWeek.id);
	}

	async dropOnWeek(weekId: string): Promise<void> {
		const dragId = this.draggingTaskId;
		const sourceWeek = dragId ? this.weekOfTask(dragId) : null;
		const targetWeek = this.weekById(weekId);
		this.endTaskDrag();
		if (!dragId || !sourceWeek || !targetWeek) return;
		if (sourceWeek.id !== targetWeek.id) {
			await this.moveTaskToWeek(dragId, targetWeek.id);
			return;
		}
		const found = locate(sourceWeek.tasks, dragId);
		if (!found) return;
		if (!found.parent && found.index === found.siblings.length - 1) return;
		const task = removeTask(sourceWeek.tasks, dragId);
		if (!task) return;
		sourceWeek.tasks.push(task);
		task.updatedAt = nowISO();
		recomputeParents(sourceWeek.tasks);
		this.markDirty(sourceWeek.id);
	}

	openSettings(section = 'ai'): void {
		this.settingsSection = section;
		this.settingsOpen = true;
	}

	focusQuickAdd(): void {
		this.quickAddFocus++;
	}
}

export const app = new AppState();
