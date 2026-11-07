<script lang="ts">
	import CalendarDays from '@lucide/svelte/icons/calendar-days';
	import CornerDownLeft from '@lucide/svelte/icons/corner-down-left';
	import Plus from '@lucide/svelte/icons/plus';
	import { app } from '#lib/app.svelte';
	import { t } from '#lib/i18n.svelte';
	import { daysOfWeek, weekdayLabel } from '#lib/date';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Popover from '#lib/components/ui/popover/index.js';

	let { weekId = '', compact = false }: { weekId?: string; compact?: boolean } = $props();

	let draft = $state('');
	let inputEl = $state<HTMLInputElement | null>(null);
	let dayOpen = $state(false);
	let dayOverride = $state<{ weekId: string; day: string | null } | null>(null);

	const week = $derived(app.weekById(weekId || app.anchorId) ?? app.week);
	const isCurrent = $derived(week ? app.isCurrent(week) : false);
	const days = $derived(week ? daysOfWeek(week.start) : []);
	const selectedDay = $derived(
		week && dayOverride?.weekId === week.id ? dayOverride.day : isCurrent ? app.today : null
	);
	const dayLabel = $derived(
		selectedDay ? (selectedDay === app.today ? t('今天') : weekdayLabel(selectedDay)) : t('未安排')
	);

	function chooseDay(day: string | null): void {
		if (!week) return;
		dayOverride = { weekId: week.id, day };
	}

	$effect(() => {
		if (app.quickAddFocus > 0 && !compact) {
			inputEl?.focus();
		}
	});

	function submit(): void {
		const text = draft.trim();
		if (!text) {
			inputEl?.focus();
			return;
		}
		app.addTask({ text, day: selectedDay, weekId: week?.id });
		draft = '';
		inputEl?.focus();
	}
</script>

<div
	class={compact
		? 'border-t px-2 py-2 pb-[calc(0.5rem+env(safe-area-inset-bottom))]'
		: 'sticky bottom-0 z-20 -mx-4 mt-6 border-t bg-background/90 px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur sm:-mx-6 sm:px-6 sm:pb-3'}
>
	<form
		class={compact ? 'flex items-center gap-1.5' : 'mx-auto flex w-full max-w-6xl items-center gap-2'}
		onsubmit={(event) => {
			event.preventDefault();
			submit();
		}}
	>
		<Input
			bind:ref={inputEl}
			bind:value={draft}
			placeholder={compact ? t('添加任务…') : t('添加任务，按 Enter 保存…')}
			maxlength={500}
			autocomplete="off"
			class={compact ? 'h-10 text-base sm:h-8 sm:text-xs' : 'h-11 flex-1 bg-background sm:h-9'}
		/>
		<Popover.Root bind:open={dayOpen}>
			<Popover.Trigger>
				{#snippet child({ props })}
					<Button
						{...props}
						variant="outline"
						size={compact ? 'icon-sm' : 'sm'}
						class={compact ? 'text-muted-foreground' : 'h-11 text-muted-foreground sm:h-9'}
						title={dayLabel}
					>
						<CalendarDays class="size-4" />
						{#if !compact}<span class="hidden sm:inline">{dayLabel}</span>{/if}
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
					{t('不安排日期')}
				</Button>
			</Popover.Content>
		</Popover.Root>
		<Button type="submit" size={compact ? 'icon-sm' : 'sm'} class={compact ? '' : 'h-11 sm:h-9'} title={t('添加')}>
			<Plus />
			{#if !compact}{t('添加')}{/if}
		</Button>
	</form>
	{#if !compact}
		<div
			class="mx-auto mt-1.5 hidden w-full max-w-6xl items-center gap-3 text-[11px] text-muted-foreground sm:flex"
		>
			<span class="flex items-center gap-1"><CornerDownLeft class="size-3" /> {t('Enter 新建同级')}</span>
			<span>{t('行首 Tab 缩进')}</span>
			<span>{t('Shift+Tab 提升')}</span>
			<span>{t('Alt+↑↓ 排序')}</span>
		</div>
	{/if}
</div>
