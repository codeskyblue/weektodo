const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

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
	return WEEKDAYS[parseISO(value).getDay()] ?? '';
}

export function monthDay(value: string): string {
	const date = parseISO(value);
	return `${date.getMonth() + 1}月${date.getDate()}日`;
}

export function formatRange(start: string, end: string): string {
	return `${monthDay(start)} – ${monthDay(end)}`;
}

export function formatWeekTitle(id: string): string {
	return `${weekYear(id)} 第 ${weekNumber(id)} 周`;
}

export function isSameWeek(id: string, date = new Date()): boolean {
	return id === weekIdFor(dateToISO(date));
}

export function daysOfWeek(start: string): { date: string; label: string; day: number }[] {
	return Array.from({ length: 7 }, (_, index) => {
		const date = addDays(start, index);
		return { date, label: WEEKDAYS[(index + 1) % 7] ?? '', day: index + 1 };
	});
}

export function shortDate(value: string): string {
	return value.slice(5);
}
