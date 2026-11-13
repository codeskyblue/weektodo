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
	import { openExternal } from '#lib/wails';
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

	const githubUrl = 'https://github.com/uiautodev/weektodo';

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
						<div class="truncate text-[17px] font-semibold sm:text-sm sm:font-medium">
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
				<Button
					variant="ghost"
					size="icon-sm"
					class="hidden sm:inline-flex"
					href={githubUrl}
					target="_blank"
					rel="noopener noreferrer"
					title={t('项目主页')}
					aria-label={t('项目主页')}
					onclick={(event) => {
						event.preventDefault();
						openExternal(githubUrl);
					}}
				>
					<svg viewBox="0 0 24 24" fill="currentColor" class="size-4" aria-hidden="true">
						<path
							d="M12 .5C5.65.5.5 5.65.5 12a11.5 11.5 0 0 0 7.86 10.92c.58.1.79-.25.79-.55v-2.16c-3.2.7-3.88-1.36-3.88-1.36-.53-1.33-1.28-1.69-1.28-1.69-1.05-.71.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.76 2.7 1.25 3.36.96.1-.75.4-1.25.73-1.54-2.55-.29-5.24-1.28-5.24-5.68 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.46.11-3.05 0 0 .96-.31 3.15 1.18a10.9 10.9 0 0 1 5.74 0c2.19-1.49 3.15-1.18 3.15-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.83 1.19 3.09 0 4.41-2.7 5.38-5.26 5.67.41.36.78 1.06.78 2.14v3.17c0 .3.21.66.8.55A11.5 11.5 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5Z"
						/>
					</svg>
				</Button>
			</div>
		</div>
	</div>
</header>
