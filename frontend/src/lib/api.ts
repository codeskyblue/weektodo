import type {
	CarryPreview,
	CarryResult,
	Meta,
	ReportMode,
	ReportResponse,
	AIModelsResponse,
	SettingsPublic,
	Task,
	UpdateInfo,
	UpdateProgress,
	Week,
	WeekSummary
} from './types';

const BASE = '/api';

export class ApiError extends Error {
	status: number;
	code: string;

	constructor(message: string, status: number, code = '') {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const headers = new Headers(init?.headers);
	if (init?.body && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}
	const response = await fetch(BASE + path, { ...init, headers });
	const text = await response.text();
	let parsed: unknown = null;
	if (text) {
		try {
			parsed = JSON.parse(text);
		} catch {
			parsed = null;
		}
	}
	if (!response.ok) {
		const payload = parsed as { error?: string; code?: string } | null;
		throw new ApiError(payload?.error || response.statusText || '请求失败', response.status, payload?.code || '');
	}
	return parsed as T;
}

export function getMeta(): Promise<Meta> {
	return request('/meta');
}

export function listWeeks(): Promise<{ currentWeekId: string; weeks: WeekSummary[] }> {
	return request('/weeks');
}

export function createWeek(id: string): Promise<{ week: Week }> {
	return request(`/weeks/${encodeURIComponent(id)}`, { method: 'POST' });
}

export function saveWeek(id: string, tasks: Task[]): Promise<{ week: Week }> {
	return request(`/weeks/${encodeURIComponent(id)}`, {
		method: 'PUT',
		body: JSON.stringify({ tasks })
	});
}

export function carryPreview(id: string): Promise<CarryPreview> {
	return request(`/weeks/${encodeURIComponent(id)}/carry-over`);
}

export function carryOver(id: string): Promise<CarryResult> {
	return request(`/weeks/${encodeURIComponent(id)}/carry-over`, { method: 'POST' });
}

export function getReport(id: string, mode: ReportMode): Promise<ReportResponse> {
	return request(`/weeks/${encodeURIComponent(id)}/report?mode=${mode}`);
}

export function polishReport(id: string, mode: ReportMode): Promise<{ markdown: string; mode: ReportMode }> {
	return request(`/weeks/${encodeURIComponent(id)}/report/polish`, {
		method: 'POST',
		body: JSON.stringify({ mode })
	});
}

export function getSettings(): Promise<SettingsPublic> {
	return request('/settings');
}

export interface SettingsDraft {
	baseUrl: string;
	model: string;
	apiKey: string;
	clearApiKey: boolean;
}

export function saveSettings(draft: SettingsDraft): Promise<SettingsPublic> {
	return request('/settings', {
		method: 'PUT',
		body: JSON.stringify({ ai: draft })
	});
}

export function testAI(draft: Pick<SettingsDraft, 'baseUrl' | 'model' | 'apiKey'>): Promise<{
	ok: boolean;
	message: string;
	model: string;
}> {
	return request('/settings/ai/test', {
		method: 'POST',
		body: JSON.stringify(draft)
	});
}

export function health(): Promise<{ ok: boolean }> {
	return request('/health');
}

export function checkUpdate(): Promise<UpdateInfo> {
	return request('/update/check');
}

export function applyUpdate(): Promise<{
	ok: boolean;
	installed: boolean;
	path: string;
	version: string;
	source: string;
	restarting: boolean;
}> {
	return request('/update/apply', { method: 'POST', body: JSON.stringify({}) });
}

export function updateStatus(): Promise<UpdateProgress> {
	return request('/update/status');
}

export function fetchAIModels(draft: { baseUrl: string; apiKey: string }): Promise<AIModelsResponse> {
	return request('/settings/ai/models', {
		method: 'POST',
		body: JSON.stringify(draft)
	});
}
