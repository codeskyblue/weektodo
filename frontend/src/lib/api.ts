import { i18n, t } from './i18n.svelte';
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
	headers.set('Accept-Language', i18n.language);
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
		throw new ApiError(payload?.error || response.statusText || t('请求失败'), response.status, payload?.code || '');
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

export async function getWeekOrNull(id: string): Promise<Week | null> {
	try {
		const response = await request<{ week: Week | null }>(
			`/weeks/${encodeURIComponent(id)}?optional=1`
		);
		return response.week;
	} catch (error) {
		if (error instanceof ApiError && error.status === 404) return null;
		throw error;
	}
}

export function moveTask(
	fromWeekId: string,
	taskId: string,
	toWeekId: string
): Promise<{ from: Week; to: Week }> {
	return request(`/weeks/${encodeURIComponent(fromWeekId)}/move`, {
		method: 'POST',
		body: JSON.stringify({ taskId, toWeekId })
	});
}

export function carryPreview(id: string): Promise<CarryPreview> {
	return request(`/weeks/${encodeURIComponent(id)}/carry-over`);
}

export function carryOver(id: string): Promise<CarryResult> {
	return request(`/weeks/${encodeURIComponent(id)}/carry-over`, { method: 'POST' });
}

export function getReport(id: string, mode: ReportMode): Promise<ReportResponse> {
	return request(`/weeks/${encodeURIComponent(id)}/report?mode=${mode}&lang=${i18n.language}`);
}

export function polishReport(
	id: string,
	mode: ReportMode
): Promise<{ markdown: string; mode: ReportMode }> {
	return request(`/weeks/${encodeURIComponent(id)}/report/polish`, {
		method: 'POST',
		body: JSON.stringify({ mode, lang: i18n.language })
	});
}

export function getSettings(): Promise<SettingsPublic> {
	return request('/settings');
}

export interface AIDraft {
	baseUrl: string;
	model: string;
	apiKey: string;
	clearApiKey: boolean;
}

export interface WebDAVDraft {
	url: string;
	username: string;
	password: string;
	remoteDir: string;
}

export interface SettingsDraft {
	ai?: AIDraft;
	webdav?: WebDAVDraft & { clearPassword?: boolean };
}

export interface WebDAVSyncResult {
	ok: boolean;
	weeks: number;
	tasks: number;
	remotePath: string;
	uploaded: boolean;
}

export function saveSettings(draft: SettingsDraft): Promise<SettingsPublic> {
	return request('/settings', {
		method: 'PUT',
		body: JSON.stringify(draft)
	});
}

export function testAI(draft: Pick<AIDraft, 'baseUrl' | 'model' | 'apiKey'>): Promise<{
	ok: boolean;
	message: string;
	model: string;
}> {
	return request('/settings/ai/test', {
		method: 'POST',
		body: JSON.stringify(draft)
	});
}

export function testWebDAV(draft: WebDAVDraft): Promise<{ ok: boolean }> {
	return request('/webdav/test', {
		method: 'POST',
		body: JSON.stringify(draft)
	});
}

export function syncWebDAV(draft: WebDAVDraft): Promise<WebDAVSyncResult> {
	return request('/webdav/sync', {
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

export function saveLanguage(language: 'zh' | 'en'): Promise<SettingsPublic> {
	return request('/settings', {
		method: 'PUT',
		body: JSON.stringify({ language })
	});
}

export function saveWeekSpan(weekSpan: 1 | 2 | 3): Promise<SettingsPublic> {
	return request('/settings', {
		method: 'PUT',
		body: JSON.stringify({ weekSpan })
	});
}
