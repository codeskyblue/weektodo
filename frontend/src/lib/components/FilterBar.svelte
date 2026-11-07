<script lang="ts">
	import Download from '@lucide/svelte/icons/download';
	import { app } from '#lib/app.svelte';
	import { t } from '#lib/i18n.svelte';
	import { flatten } from '#lib/tree';
	import type { Filter } from '#lib/types';
	import { Button } from '#lib/components/ui/button/index.js';
	import { ToggleGroup, ToggleGroupItem } from '#lib/components/ui/toggle-group/index.js';

	const filters: { value: Filter; label: string }[] = [
		{ value: 'all', label: '全部' },
		{ value: 'today', label: '今天' },
		{ value: 'todo', label: '未完成' },
		{ value: 'done', label: '已完成' },
		{ value: 'dropped', label: '已废弃' }
	];

	const todayTodo = $derived(
		app.week
			? flatten(app.week.tasks).filter(
					(row) =>
						row.task.children.length === 0 &&
						row.task.day === app.today &&
						row.task.status === 'todo'
				).length
			: 0
	);
</script>

<div class="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between sm:gap-3">
	<div class="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:overflow-visible sm:px-0">
		<ToggleGroup
			type="single"
			variant="outline"
			size="sm"
			bind:value={app.filter}
			aria-label={t('任务筛选')}
			class="max-sm:w-max"
		>
			{#each filters as item (item.value)}
				<ToggleGroupItem
					value={item.value}
					class="max-sm:h-9 max-sm:px-3 max-sm:text-sm">{t(item.label)}</ToggleGroupItem
				>
			{/each}
		</ToggleGroup>
	</div>
	<div
		class="flex items-center justify-between gap-2 text-xs text-muted-foreground sm:justify-end"
	>
		<span class="min-w-0 truncate">
			{#if app.filter === 'today'}
				{t('今天还有 {count} 项待办', { count: todayTodo })}
			{:else if app.stats.unplanned > 0}
				{t('{count} 项未安排日期', { count: app.stats.unplanned })}
			{:else}
				{t('所有任务都已安排日期')}
			{/if}
		</span>
		<Button
			variant="ghost"
			size="xs"
			class="shrink-0 max-sm:h-9 max-sm:px-2 max-sm:text-xs"
			disabled={app.carryBusy}
			onclick={() => app.openCarryOver()}
			title={t('把上一周未完成的任务带入本周')}
		>
			<Download class="size-3.5" /> {t('带入上周')}
		</Button>
	</div>
</div>
