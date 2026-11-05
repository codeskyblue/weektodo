export type TaskStatus = 'todo' | 'done' | 'dropped';

export interface CarriedFrom {
	weekId: string;
	taskId: string;
}

export interface Task {
	id: string;
	text: string;
	status: TaskStatus;
	day?: string;
	createdAt?: string;
	completedAt?: string;
	droppedAt?: string;
	carriedFrom?: CarriedFrom;
	children: Task[];
}

export interface Week {
	id: string;
	start: string;
	end: string;
	createdAt?: string;
	tasks: Task[];
}

export interface WeekSummary {
	id: string;
	start: string;
	end: string;
	total: number;
	done: number;
	dropped: number;
	unplanned: number;
}

export interface Stats {
	total: number;
	done: number;
	dropped: number;
	unplanned: number;
}

export interface AIPublic {
	baseUrl: string;
	model: string;
	configured: boolean;
	apiKeyHint: string;
}

export interface SettingsPublic {
	ai: AIPublic;
}

export interface Meta {
	dataDir: string;
	version: string;
	commit: string;
	currentWeekId: string;
}

export interface CarryItem {
	id: string;
	text: string;
	depth: number;
}

export interface CarryPreview {
	sourceWeekId: string;
	count: number;
	skipped: number;
	tasks: CarryItem[];
	reason?: string;
}

export interface CarryResult {
	week: Week;
	created: number;
	skipped: number;
	reason?: string;
}

export type ReportMode = 'brief' | 'full';

export interface ReportResponse {
	markdown: string;
	stats: Stats;
	mode: ReportMode;
}

export type Filter = 'all' | 'today' | 'todo' | 'done' | 'dropped';

export interface TaskRow {
	task: Task;
	depth: number;
	parent: Task | null;
	siblings: Task[];
	index: number;
	context: boolean;
}

export interface ConfirmOptions {
	title: string;
	description?: string;
	confirmLabel?: string;
	destructive?: boolean;
	onConfirm?: () => void | Promise<void>;
}

export interface UpdateInfo {
	current: string;
	latest: string;
	available: boolean;
	dev: boolean;
	release_url: string;
	notes: string;
	published_at: string;
	asset?: string;
	asset_size?: number;
	source?: string;
	desktop: boolean;
}

export interface UpdateProgress {
	active: boolean;
	stage: string;
	downloaded: number;
	total: number;
	percent: number;
}
