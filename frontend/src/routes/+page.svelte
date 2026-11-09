<script lang="ts">
	import { onMount } from 'svelte';
	import { app } from '#lib/app.svelte';
	import { t } from '#lib/i18n.svelte';
	import AppHeader from '#lib/components/AppHeader.svelte';
	import ConfirmDialog from '#lib/components/ConfirmDialog.svelte';
	import EmptyState from '#lib/components/EmptyState.svelte';
	import FilterBar from '#lib/components/FilterBar.svelte';
	import QuickAdd from '#lib/components/QuickAdd.svelte';
	import ReportSheet from '#lib/components/ReportSheet.svelte';
	import SettingsDialog from '#lib/components/SettingsDialog.svelte';
	import TaskList from '#lib/components/TaskList.svelte';
	import WeekColumn from '#lib/components/WeekColumn.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card, CardContent } from '#lib/components/ui/card/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';

	const rows = $derived(app.rows);

	function handleGlobalKeydown(event: KeyboardEvent): void {
		// Cmd+, on macOS and Ctrl+, elsewhere opens settings, matching the
		// platform convention. Let the IME keep its own key handling.
		if (event.isComposing) return;
		if (!(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) return;
		if (event.key !== ',') return;
		event.preventDefault();
		if (!app.loading && !app.settingsOpen) app.openSettings();
	}

	onMount(() => {
		void app.init();
	});
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<div class="flex min-h-svh flex-col bg-background">
	<AppHeader />

	{#if app.loading}
		<main class="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:px-6">
			<div class="space-y-4">
				<div class="flex gap-2">
					{#each Array(5) as _, index (index)}
						<Skeleton class="h-7 w-16" />
					{/each}
				</div>
				<div class="space-y-2">
					{#each Array(8) as _, index (index)}
						<Skeleton class="h-9 w-full" />
					{/each}
				</div>
			</div>
		</main>
	{:else if app.error}
		<main class="mx-auto flex w-full max-w-6xl flex-1 items-center justify-center px-4 py-10 sm:px-6">
			<Card class="w-full max-w-md">
				<CardContent class="flex flex-col items-center gap-3 py-10 text-center">
					<div class="text-sm font-medium">{t('加载失败')}</div>
					<p class="text-xs text-muted-foreground">{app.error}</p>
					<Button variant="outline" size="sm" onclick={() => app.init()}>{t('重试')}</Button>
				</CardContent>
			</Card>
		</main>
	{:else if app.week}
		<main
			class="mx-auto flex w-full flex-1 flex-col px-4 sm:px-6"
			class:max-w-6xl={app.weekSpan === 1}
			class:max-w-7xl={app.weekSpan > 1}
		>
			<div class="pt-4">
				<FilterBar />
			</div>
			{#if app.weekSpan === 1}
				<div
					role="group"
					class="flex-1 rounded-xl py-4 transition-colors {app.dropWeekId === app.anchorId
						? 'bg-primary/5 ring-2 ring-primary/40 ring-inset'
						: ''}"
					ondragover={(event) => {
						if (!app.draggingTaskId) return;
						event.preventDefault();
						if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
						app.setWeekDropTarget(app.anchorId);
					}}
					ondragleave={(event) => {
						const related = event.relatedTarget as Node | null;
						if (related && event.currentTarget.contains(related)) return;
						app.clearWeekDropTarget(app.anchorId);
					}}
					ondrop={(event) => {
						event.preventDefault();
						void app.dropOnWeek(app.anchorId);
					}}
				>
					{#if rows.length === 0}
						<EmptyState filtered={app.week.tasks.length > 0} />
					{:else}
						<TaskList {rows} />
					{/if}
				</div>
				<QuickAdd />
			{:else}
				<div
					class="grid flex-1 items-start gap-4 py-4 {app.weekSpan === 3
						? 'lg:grid-cols-3'
						: 'lg:grid-cols-2'}"
				>
					{#each app.weeks as week (week.id)}
						<WeekColumn {week} />
					{/each}
				</div>
			{/if}
		</main>
	{/if}
</div>

<ConfirmDialog />
<ReportSheet />
{#if app.settingsOpen}<SettingsDialog />{/if}
