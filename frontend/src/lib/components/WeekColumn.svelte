<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import { app } from '#lib/app.svelte';
	import { t } from '#lib/i18n.svelte';
	import { formatRange, formatWeekTitle } from '#lib/date';
	import type { Week } from '#lib/types';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Progress } from '#lib/components/ui/progress/index.js';
	import QuickAdd from './QuickAdd.svelte';
	import TaskList from './TaskList.svelte';

	let { week }: { week: Week } = $props();

	const rows = $derived(app.rowsFor(week));
	const stats = $derived(app.statsFor(week));
	const progress = $derived(app.progressFor(week));
</script>

<section class="flex min-w-0 flex-col rounded-xl border bg-background shadow-xs">
	<header class="flex items-start justify-between gap-2 border-b px-3 py-2.5">
		<div class="min-w-0">
			<div class="flex min-w-0 items-center gap-1.5">
				<h2 class="truncate text-sm font-medium">{formatWeekTitle(week.id)}</h2>
				{#if app.isCurrent(week)}
					<Badge variant="secondary" class="shrink-0 px-1.5 py-0 text-[10px]">{t('本周')}</Badge>
				{/if}
			</div>
			<div class="mt-0.5 truncate text-xs text-muted-foreground">
				{formatRange(week.start, week.end)}
			</div>
		</div>
		<div class="shrink-0 text-right">
			<div class="text-xs tabular-nums text-muted-foreground">{stats.done}/{stats.total}</div>
			<Progress value={progress} class="mt-1 w-12" />
		</div>
	</header>

	<div class="min-h-32 flex-1 px-1.5 py-2 sm:min-h-48">
		{#if rows.length === 0}
			<div class="flex flex-col items-center gap-2 px-3 py-8 text-center">
				<p class="text-xs text-muted-foreground">
					{week.tasks.length > 0 ? t('没有符合筛选的任务') : t('这一周还没有任务')}
				</p>
				{#if week.tasks.length === 0 && app.filter === 'all'}
					<Button variant="outline" size="xs" onclick={() => app.addTask({ weekId: week.id })}>
						<Plus class="size-3" /> {t('添加第一条')}
					</Button>
				{/if}
			</div>
		{:else}
			<TaskList {rows} />
		{/if}
	</div>

	<QuickAdd weekId={week.id} compact />
</section>
