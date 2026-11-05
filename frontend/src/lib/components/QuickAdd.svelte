<script lang="ts">
	import CalendarDays from '@lucide/svelte/icons/calendar-days';
	import CornerDownLeft from '@lucide/svelte/icons/corner-down-left';
	import Plus from '@lucide/svelte/icons/plus';
	import { app } from '#lib/app.svelte';
	import { daysOfWeek, weekdayLabel } from '#lib/date';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Popover from '#lib/components/ui/popover/index.js';

	let draft = $state('');
	let inputEl = $state<HTMLInputElement | null>(null);
	let dayOpen = $state(false);
	let dayOverride = $state<{ weekId: string; day: string | null } | null>(null);

	const days = $derived(app.week ? daysOfWeek(app.week.start) : []);
	const selectedDay = $derived(
		app.week && dayOverride?.weekId === app.week.id
			? dayOverride.day
			: app.isCurrentWeek
				? app.today
				: null
	);
	const dayLabel = $derived(
		selectedDay ? (selectedDay === app.today ? '今天' : weekdayLabel(selectedDay)) : '未安排'
	);

	function chooseDay(day: string | null): void {
		if (!app.week) return;
		dayOverride = { weekId: app.week.id, day };
	}

	$effect(() => {
		if (app.quickAddFocus > 0) {
			inputEl?.focus();
		}
	});

	function submit(): void {
		const text = draft.trim();
		if (!text) {
			inputEl?.focus();
			return;
		}
		app.addTask({ text, day: selectedDay });
		draft = '';
		inputEl?.focus();
	}
</script>

<div class="sticky bottom-0 z-20 -mx-4 mt-6 border-t bg-background/90 px-4 py-3 backdrop-blur sm:-mx-6 sm:px-6">
	<form class="mx-auto flex w-full max-w-4xl items-center gap-2" onsubmit={(event) => { event.preventDefault(); submit(); }}>
		<Input
			bind:ref={inputEl}
			bind:value={draft}
			placeholder="添加任务，按 Enter 保存…"
			maxlength={500}
			autocomplete="off"
			class="h-9 flex-1 bg-background"
		/>
		<Popover.Root bind:open={dayOpen}>
			<Popover.Trigger>
				{#snippet child({ props })}
					<Button {...props} variant="outline" size="sm" class="h-9 text-muted-foreground">
						<CalendarDays class="size-4" />
						<span class="hidden sm:inline">{dayLabel}</span>
					</Button>
				{/snippet}
			</Popover.Trigger>
			<Popover.Content align="end" class="w-auto p-2">
				<div class="grid grid-cols-7 gap-1">
					{#each days as day (day.date)}
						<button
							type="button"
							class="flex w-11 flex-col items-center gap-0.5 rounded-md px-1 py-1.5 text-xs transition-colors hover:bg-muted {selectedDay ===
							day.date
								? 'bg-secondary font-medium text-secondary-foreground'
								: 'text-muted-foreground'}"
							onclick={() => {
								chooseDay(day.date);
								dayOpen = false;
								inputEl?.focus();
							}}
						>
							<span>{day.label}</span>
							<span class="text-[10px] tabular-nums opacity-70">{day.date.slice(8)}</span>
						</button>
					{/each}
				</div>
				<Button
					variant="ghost"
					size="xs"
					class="mt-1 w-full justify-start text-muted-foreground"
					onclick={() => {
						chooseDay(null);
						dayOpen = false;
						inputEl?.focus();
					}}
				>
					不安排日期
				</Button>
			</Popover.Content>
		</Popover.Root>
		<Button type="submit" size="sm" class="h-9">
			<Plus /> 添加
		</Button>
	</form>
	<div class="mx-auto mt-1.5 hidden w-full max-w-4xl items-center gap-3 text-[11px] text-muted-foreground sm:flex">
		<span class="flex items-center gap-1"><CornerDownLeft class="size-3" /> Enter 新建同级</span>
		<span>行首 Tab 缩进</span>
		<span>Shift+Tab 提升</span>
		<span>Alt+↑↓ 排序</span>
	</div>
</div>
