<script lang="ts">
	import Bot from '@lucide/svelte/icons/bot';
	import Copy from '@lucide/svelte/icons/copy';
	import Download from '@lucide/svelte/icons/download';
	import Loader2 from '@lucide/svelte/icons/loader-2';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Sparkles from '@lucide/svelte/icons/sparkles';
	import DOMPurify from 'dompurify';
	import { marked } from 'marked';
	import { toast } from 'svelte-sonner';
	import { app } from '#lib/app.svelte';
	import { t } from '#lib/i18n.svelte';
	import { formatRange, formatWeekTitle } from '#lib/date';
	import type { ReportMode } from '#lib/types';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';

	const html = $derived(DOMPurify.sanitize(marked.parse(app.reportText) as string));

	async function copyReport(): Promise<void> {
		const text = app.reportText;
		if (!text) return;
		try {
			await navigator.clipboard.writeText(text);
		} catch {
			const area = document.createElement('textarea');
			area.value = text;
			area.style.position = 'fixed';
			area.style.opacity = '0';
			document.body.appendChild(area);
			area.select();
			document.execCommand('copy');
			area.remove();
		}
		toast.success(t('周报已复制'));
	}

	function downloadReport(): void {
		const text = app.reportText;
		if (!text || !app.week) return;
		const blob = new Blob([text], { type: 'text/markdown;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const anchor = document.createElement('a');
		anchor.href = url;
		anchor.download = t('周报-{id}.md', { id: app.week.id });
		anchor.click();
		URL.revokeObjectURL(url);
	}
</script>

<Sheet.Root bind:open={app.reportOpen}>
	<Sheet.Content side="right" class="w-full gap-0 p-0 sm:max-w-2xl">
		<div class="border-b px-5 py-4 pr-12">
			<Sheet.Header class="gap-1 p-0">
				<Sheet.Title class="flex items-center gap-2 text-base">
					{t('周报')}
					{#if app.aiMarkdown}
						<Badge variant="secondary" class="gap-1">
							<Bot class="size-3" /> {t('AI 润色')}
						</Badge>
					{/if}
				</Sheet.Title>
				<Sheet.Description>
					{#if app.week}
						{formatWeekTitle(app.week.id)} · {formatRange(app.week.start, app.week.end)}
					{/if}
				</Sheet.Description>
			</Sheet.Header>
			<div class="mt-3 flex flex-wrap items-center gap-1.5">
				<Badge variant="outline">{t('完成 {count}', { count: app.reportStats.done })}</Badge>
				<Badge variant="outline">{t('废弃 {count}', { count: app.reportStats.dropped })}</Badge>
				<Badge variant="outline">
					{t('未完成 {count}', { count: Math.max(0, app.reportStats.total - app.reportStats.done) })}
				</Badge>
				<div class="ml-auto flex items-center gap-1">
					<Button
						variant="ghost"
						size="icon-sm"
						class="max-sm:size-10"
						onclick={copyReport}
						disabled={!app.reportText}
						title={t('复制 Markdown')}
					>
						<Copy />
					</Button>
					<Button
						variant="ghost"
						size="icon-sm"
						class="max-sm:size-10"
						onclick={downloadReport}
						disabled={!app.reportText}
						title={t('下载 Markdown')}
					>
						<Download />
					</Button>
				</div>
			</div>
		</div>

		<div class="flex flex-wrap items-center justify-between gap-2 border-b px-5 py-2.5">
			<Tabs.Root
				value={app.reportMode}
				onValueChange={(value) => app.setReportMode(value as ReportMode)}
			>
				<Tabs.List>
					<Tabs.Trigger value="brief">{t('精简')}</Tabs.Trigger>
					<Tabs.Trigger value="full">{t('完整')}</Tabs.Trigger>
				</Tabs.List>
			</Tabs.Root>
			<div class="flex items-center gap-1.5">
				{#if app.aiMarkdown}
					<Button variant="ghost" size="sm" onclick={() => app.clearAIReport()}>
						<RotateCcw /> {t('恢复本地')}
					</Button>
				{/if}
				<Button
					variant="outline"
					size="sm"
					class="max-sm:h-9"
					onclick={() => app.polishReport()}
					disabled={app.aiLoading || app.reportLoading || !app.reportMarkdown}
				>
					{#if app.aiLoading}
						<Loader2 class="animate-spin" /> {t('生成中…')}
					{:else}
						<Sparkles /> {t('AI 润色')}
					{/if}
				</Button>
			</div>
		</div>

		<div class="flex-1 overflow-y-auto px-5 py-4 max-sm:pb-[calc(1rem+env(safe-area-inset-bottom))]">
			{#if app.reportLoading}
				<div class="space-y-3">
					<Skeleton class="h-5 w-2/3" />
					<Skeleton class="h-4 w-full" />
					<Skeleton class="h-4 w-5/6" />
					<Skeleton class="h-4 w-4/5" />
					<Skeleton class="h-4 w-2/3" />
				</div>
			{:else if app.reportError}
				<div class="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
					{app.reportError}
				</div>
			{:else if app.reportText}
				<div class="markdown-body">
					{@html html}
				</div>
			{:else}
				<p class="text-sm text-muted-foreground">{t('本周还没有可以写进周报的内容。')}</p>
			{/if}
			{#if app.aiMarkdown}
				<Separator class="my-5" />
				<p class="text-xs text-muted-foreground">
					{t('AI 生成的内容可能有偏差，发送前请检查。')}
				</p>
			{/if}
		</div>
	</Sheet.Content>
</Sheet.Root>
