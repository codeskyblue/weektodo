<script lang="ts">
	import Download from '@lucide/svelte/icons/download';
	import { app } from '#lib/app.svelte';
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

<div class="flex flex-wrap items-center justify-between gap-3">
	<ToggleGroup
		type="single"
		variant="outline"
		size="sm"
		bind:value={app.filter}
		aria-label="任务筛选"
	>
		{#each filters as item (item.value)}
			<ToggleGroupItem value={item.value}>{item.label}</ToggleGroupItem>
		{/each}
	</ToggleGroup>
	<div class="flex items-center gap-2 text-xs text-muted-foreground">
		{#if app.filter === 'today'}
			今天还有 {todayTodo} 项待办
		{:else if app.stats.unplanned > 0}
			{app.stats.unplanned} 项未安排日期
		{:else}
			所有任务都已安排日期
		{/if}
		<Button
			variant="ghost"
			size="xs"
			disabled={app.carryBusy}
			onclick={() => app.openCarryOver()}
			title="把上一周未完成的任务带入本周"
		>
			<Download class="size-3.5" /> 带入上周
		</Button>
	</div>
</div>
