<script lang="ts">
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import FileText from '@lucide/svelte/icons/file-text';
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
</script>

<header class="sticky top-0 z-30 border-b bg-background/85 backdrop-blur">
	<div class="mx-auto w-full max-w-4xl px-4 py-3 sm:px-6">
		<div class="flex items-center gap-3">
			<button
				type="button"
				class="flex min-w-0 items-center gap-2.5 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
				onclick={() => app.goToday()}
				aria-label={t('WeekTodo 首页')}
				title={t('回到本周')}
			>
				<Logo size={30} />
				<span class="hidden text-[15px] font-semibold tracking-tight sm:block">WeekTodo</span>
			</button>

			<div class="ml-1 flex min-w-0 items-center gap-1">
				<Button
					variant="ghost"
					size="icon-sm"
					onclick={previous}
					disabled={app.navigating || !app.week}
					title={t('上一周')}
				>
					<ChevronLeft />
				</Button>
				<div class="min-w-0 px-1 text-center sm:px-2">
					{#if app.week}
						<div class="truncate text-sm font-medium">{formatWeekTitle(app.week.id)}</div>
						<div class="truncate text-xs text-muted-foreground">
							{formatRange(app.week.start, app.week.end)}
						</div>
					{:else}
						<div class="text-sm text-muted-foreground">{t('加载中…')}</div>
					{/if}
				</div>
				<Button
					variant="ghost"
					size="icon-sm"
					onclick={next}
					disabled={app.navigating || !app.week}
					title={t('下一周')}
				>
					<ChevronRight />
				</Button>
				{#if app.week && !app.isCurrentWeek}
					<Button variant="outline" size="xs" class="ml-1" onclick={() => app.goToday()}>
						{t('回到本周')}
					</Button>
				{/if}
			</div>

			<div class="ml-auto flex items-center gap-1.5">
				<div class="mr-1 hidden items-center gap-2 sm:flex">
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
				<Button variant="outline" size="sm" onclick={() => app.openReport()}>
					<FileText />
					<span class="hidden sm:inline">{t('周报')}</span>
				</Button>
				<Button
					variant="ghost"
					size="icon-sm"
					onclick={() => (app.settingsOpen = true)}
					title={t('设置')}
				>
					<Settings />
				</Button>
				<Button
					variant="ghost"
					size="icon-sm"
					onclick={() => app.setLanguage(i18n.language === 'zh' ? 'en' : 'zh')}
					title={t('切换语言')}
					aria-label={t('切换语言')}
				>
					<span class="text-[15px] font-semibold leading-none">文</span>
				</Button>
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
</header>
