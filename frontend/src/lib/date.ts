import { i18n, t } from './i18n.svelte';

function locale(): string {
	return i18n.language === 'en' ? 'en-US' : 'zh-CN';
}

export function pad(value: number): string {
	return String(value).padStart(2, '0');
}

export function dateToISO(date: Date): string {
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

export function todayISO(): string {
	return dateToISO(new Date());
}

export function parseISO(value: string): Date {
	const [year, month, day] = value.split('-').map(Number);
	return new Date(year, (month ?? 1) - 1, day ?? 1);
}

export function addDays(value: string, amount: number): string {
	const date = parseISO(value);
	date.setDate(date.getDate() + amount);
	return dateToISO(date);
}

export function weekStartOf(value: string): string {
	const date = parseISO(value);
	const offset = (date.getDay() + 6) % 7;
	date.setDate(date.getDate() - offset);
	return dateToISO(date);
}

export function weekIdFor(value: string): string {
	const date = parseISO(value);
	const target = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
	const dayNum = target.getUTCDay() || 7;
	target.setUTCDate(target.getUTCDate() + 4 - dayNum);
	const year = target.getUTCFullYear();
	const yearStart = new Date(Date.UTC(year, 0, 1));
	const week = Math.ceil(((target.getTime() - yearStart.getTime()) / 86400000 + 1) / 7);
	return `${year}-W${pad(week)}`;
}

export function previousWeekId(id: string): string {
	const start = weekStart(id);
	return weekIdFor(addDays(start, -7));
}

export function nextWeekId(id: string): string {
	const start = weekStart(id);
	return weekIdFor(addDays(start, 7));
}

export function weekStart(id: string): string {
	const [year, week] = id.split('-W').map(Number);
	return weekStartFor(year, week);
}

function weekStartFor(year: number, week: number): string {
	const jan4 = new Date(year, 0, 4);
	const offset = (jan4.getDay() + 6) % 7;
	jan4.setDate(jan4.getDate() - offset + (week - 1) * 7);
	return dateToISO(jan4);
}

export function weekNumber(id: string): number {
	return Number(id.split('-W')[1] ?? 0);
}

export function weekYear(id: string): number {
	return Number(id.split('-W')[0] ?? 0);
}

export function weekdayLabel(value: string): string {
	return new Intl.DateTimeFormat(locale(), { weekday: 'short' }).format(parseISO(value));
}

export function monthDay(value: string): string {
	return new Intl.DateTimeFormat(locale(), { month: 'short', day: 'numeric' }).format(parseISO(value));
}

export function formatRange(start: string, end: string): string {
	return `${monthDay(start)} – ${monthDay(end)}`;
}

export function formatWeekTitle(id: string): string {
	return t('{year} 第 {week} 周', { year: weekYear(id), week: weekNumber(id) });
}

export function formatWeekTitleCompact(id: string): string {
	return `${weekYear(id)}-W${String(weekNumber(id)).padStart(2, '0')}`;
}

export function formatRangeCompact(start: string, end: string): string {
	const monthDay = (value: string) => `${Number(value.slice(5, 7))}.${Number(value.slice(8, 10))}`;
	return `${monthDay(start)} – ${monthDay(end)}`;
}

export function isSameWeek(id: string, date = new Date()): boolean {
	return id === weekIdFor(dateToISO(date));
}

export function daysOfWeek(start: string): { date: string; label: string; day: number }[] {
	return Array.from({ length: 7 }, (_, index) => {
		const date = addDays(start, index);
		return { date, label: weekdayLabel(date), day: index + 1 };
	});
}

export function shortDate(value: string): string {
	return value.slice(5);
}
