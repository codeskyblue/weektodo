<script lang="ts">
	import CalendarDays from '@lucide/svelte/icons/calendar-days';
	import X from '@lucide/svelte/icons/x';
	import { app } from '#lib/app.svelte';
	import { daysOfWeek, weekdayLabel } from '#lib/date';
	import type { Task } from '#lib/types';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Popover from '#lib/components/ui/popover/index.js';

	let { task }: { task: Task } = $props();
	let open = $state(false);

	const days = $derived(app.week ? daysOfWeek(app.week.start) : []);
	const label = $derived(task.day ? (task.day === app.today ? '今天' : weekdayLabel(task.day)) : '');
</script>

<Popover.Root bind:open>
	<Popover.Trigger>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="ghost"
				size="xs"
				disabled={task.status === 'dropped'}
				class="h-6 gap-1 px-1.5 text-xs font-normal text-muted-foreground hover:text-foreground"
				title="设置日期"
			>
				<CalendarDays class="size-3.5" />
				{#if label}<span>{label}</span>{/if}
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content align="start" class="w-auto p-2">
		<div class="grid grid-cols-7 gap-1">
			{#each days as day (day.date)}
				<button
					type="button"
					class="flex w-11 flex-col items-center gap-0.5 rounded-md px-1 py-1.5 text-xs transition-colors hover:bg-muted {task.day ===
					day.date
						? 'bg-secondary font-medium text-secondary-foreground'
						: 'text-muted-foreground'}"
					onclick={() => {
						app.setDay(task.id, day.date);
						open = false;
					}}
				>
					<span>{day.label}</span>
					<span class="text-[10px] tabular-nums opacity-70">{day.date.slice(8)}</span>
				</button>
			{/each}
		</div>
		{#if task.day}
			<Button
				variant="ghost"
				size="xs"
				class="mt-1 w-full justify-start text-muted-foreground"
				onclick={() => {
					app.setDay(task.id, null);
					open = false;
				}}
			>
				<X class="size-3.5" /> 清除日期
			</Button>
		{/if}
	</Popover.Content>
</Popover.Root>
