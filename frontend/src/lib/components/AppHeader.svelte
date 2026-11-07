<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import FileText from '@lucide/svelte/icons/file-text';
	import Languages from '@lucide/svelte/icons/languages';
	import Monitor from '@lucide/svelte/icons/monitor';
	import Moon from '@lucide/svelte/icons/moon';
	import Settings from '@lucide/svelte/icons/settings';
	import Sun from '@lucide/svelte/icons/sun';
	import { setMode, userPrefersMode } from 'mode-watcher';
	import { app } from '#lib/app.svelte';
	import { i18n, t } from '#lib/i18n.svelte';
	import { formatRange, formatWeekTitle } from '#lib/date';
	import Logo from './Logo.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Progress } from '#lib/components/ui/progress/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';

	async function previous(): Promise<void> {
		await app.goRelative(-1);
	}

	async function next(): Promise<void> {
		await app.goRelative(1);
	}

	const versionLabel = $derived.by(() => {
		const version = app.meta?.version?.trim();
		if (!version || version === 'dev' || version === 'unknown') return 'dev';
		return version.startsWith('v') ? version : `v${version}`;
	});

	const versionTitle = $derived.by(() => {
		const parts = ['WeekTodo', app.meta?.version || 'dev'];
		if (app.meta?.commit && app.meta.commit !== 'unknown') parts.push(app.meta.commit.slice(0, 7));
		return parts.join(' · ');
	});
</script>

<header
	class="sticky top-0 z-30 border-b bg-background/90 pt-[env(safe-area-inset-top)] backdrop-blur"
>
	<div
		class="mx-auto w-full px-4 py-3 sm:px-6"
		class:max-w-6xl={app.weekSpan === 1}
		class:max-w-7xl={app.weekSpan > 1}
	>
		<div class="flex flex-wrap items-center gap-x-2 gap-y-1.5 sm:flex-nowrap sm:gap-3">
			<button
				type="button"
				class="order-1 flex size-10 shrink-0 items-center justify-center rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-ring sm:size-auto sm:justify-start sm:gap-2.5 sm:rounded-md"
				onclick={() => app.goToday()}
				aria-label={t('WeekTodo 首页')}
				title={t('回到本周')}
			>
				<Logo size={30} />
				<span class="hidden text-[15px] font-semibold tracking-tight lg:block">WeekTodo</span>
			</button>
			<button
				type="button"
				class="order-1 hidden shrink-0 rounded px-1 py-0.5 text-[11px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground lg:block"
				title={versionTitle}
				onclick={() => app.openSettings('update')}
			>
				{versionLabel}
			</button>

			<div class="order-2 flex min-w-0 items-center">
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button
								{...props}
								variant="ghost"
								size="sm"
								class="h-10 max-w-44 gap-1.5 rounded-full border border-border/70 px-3 sm:h-7 sm:max-w-36 sm:rounded-md sm:border-0 sm:px-2"
							>
								<span
									class="size-2.5 shrink-0 rounded-full"
									style={`background-color: ${app.activeCategory?.color || '#3b82f6'}`}
								></span>
								<span class="truncate text-sm font-medium sm:text-xs">
									{app.activeCategory?.name ?? t('分类')}
								</span>
								<ChevronDown class="size-3.5 shrink-0 text-muted-foreground sm:hidden" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="start" class="w-52">
						{#each app.activeCategories as category (category.id)}
							<DropdownMenu.Item onSelect={() => app.switchCategory(category.id)}>
								<span
									class="size-2.5 rounded-full"
									style={`background-color: ${category.color || '#3b82f6'}`}
								></span>
								<span class="truncate">{category.name}</span>
								{#if category.id === app.activeCategoryId}
									<Check class="ml-auto size-3.5" />
								{/if}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>

			<div
				class="order-4 flex w-full min-w-0 items-center gap-1 sm:order-3 sm:ml-1 sm:w-auto"
			>
				<Button
					variant="ghost"
					size="icon-sm"
					class="size-10 sm:size-7"
					onclick={previous}
					disabled={app.navigating || !app.week}
					title={t('上一周')}
				>
					<ChevronLeft />
				</Button>
				<button
					type="button"
					class="min-w-0 flex-1 rounded-lg px-2 py-1 text-center outline-none transition-colors hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring sm:flex-none sm:hover:bg-transparent"
					onclick={() => app.goToday()}
					title={t('回到本周')}
				>
					{#if app.week}
						<div class="truncate text-[15px] font-semibold sm:text-sm sm:font-medium">
							{formatWeekTitle(app.week.id)}
						</div>
						<div class="truncate text-xs text-muted-foreground">
							{formatRange(app.week.start, app.week.end)}
							{#if app.stats.total > 0}
								<span class="tabular-nums sm:hidden"> · {app.stats.done}/{app.stats.total}</span>
							{/if}
						</div>
					{:else}
						<div class="text-sm text-muted-foreground">{t('加载中…')}</div>
					{/if}
				</button>
				<Button
					variant="ghost"
					size="icon-sm"
					class="size-10 sm:size-7"
					onclick={next}
					disabled={app.navigating || !app.week}
					title={t('下一周')}
				>
					<ChevronRight />
				</Button>
				{#if app.week && !app.isCurrentWeek}
					<Button
						variant="outline"
						size="xs"
						class="h-9 px-2.5 sm:ml-1 sm:h-6"
						onclick={() => app.goToday()}
					>
						<span class="sm:hidden">{t('本周')}</span>
						<span class="hidden sm:inline">{t('回到本周')}</span>
					</Button>
				{/if}
			</div>

			<div class="order-3 ml-auto flex items-center gap-1.5 sm:order-4">
				<div class="mr-1 hidden items-center gap-2 md:flex">
					{#if app.saveStatus === 'saving'}
						<span class="text-xs text-muted-foreground">{t('保存中…')}</span>
					{:else if app.saveStatus === 'error'}
						<button
							type="button"
							class="text-xs text-destructive hover:underline"
							onclick={() => app.retrySave()}
							title={app.saveError}
						>
							{t('保存失败，重试')}
						</button>
					{/if}
					<span class="text-xs tabular-nums text-muted-foreground">
						{app.stats.done}/{app.stats.total}
					</span>
					<Progress value={app.progress} class="w-20" />
				</div>
				<div
					class="hidden items-center rounded-md border bg-background p-0.5 sm:flex"
					role="group"
					aria-label={t('显示周数')}
				>
					{#each [1, 2, 3] as count (count)}
						<button
							type="button"
							class="min-w-6 whitespace-nowrap rounded px-1.5 py-1 text-xs tabular-nums text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
							class:bg-muted={app.weekSpan === count}
							class:font-medium={app.weekSpan === count}
							class:text-foreground={app.weekSpan === count}
							onclick={() => app.setWeekSpan(count as 1 | 2 | 3)}
							title={t('显示 {count} 周', { count })}
							aria-label={t('{count}周', { count })}
							aria-pressed={app.weekSpan === count}
						>
							<span class="hidden lg:inline">{t('{count}周', { count })}</span>
							<span class="lg:hidden">{count}</span>
						</button>
					{/each}
				</div>
				<Button
					variant="outline"
					size="sm"
					class="max-sm:size-10 max-sm:rounded-xl sm:rounded-lg"
					onclick={() => app.openReport()}
					aria-label={t('周报')}
				>
					<FileText />
					<span class="hidden sm:inline">{t('周报')}</span>
				</Button>
				<Button
					variant="ghost"
					size="icon-sm"
					class="max-sm:size-10 max-sm:rounded-xl"
					onclick={() => app.openSettings('ai')}
					title={t('设置')}
					aria-label={t('设置')}
				>
					<Settings />
				</Button>
				<Button
					variant="ghost"
					size="icon-sm"
					class="hidden sm:inline-flex"
					onclick={() => app.setLanguage(i18n.language === 'zh' ? 'en' : 'zh')}
					title={t('切换语言')}
					aria-label={t('切换语言')}
				>
					<Languages />
				</Button>
				<div class="hidden sm:block">
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Button variant="ghost" size="icon-sm" title={t('切换主题')} {...props}>
									{#if userPrefersMode.current === 'dark'}
										<Moon />
									{:else if userPrefersMode.current === 'light'}
										<Sun />
									{:else}
										<Monitor />
									{/if}
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end" class="w-36">
							<DropdownMenu.Item onSelect={() => setMode('light')}>
								<Sun class="size-4" /> {t('浅色')}
							</DropdownMenu.Item>
							<DropdownMenu.Item onSelect={() => setMode('dark')}>
								<Moon class="size-4" /> {t('深色')}
							</DropdownMenu.Item>
							<DropdownMenu.Item onSelect={() => setMode('system')}>
								<Monitor class="size-4" /> {t('跟随系统')}
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</div>
			</div>
		</div>
	</div>
</header>
