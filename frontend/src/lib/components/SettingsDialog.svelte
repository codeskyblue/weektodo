<script lang="ts">
	import { onDestroy } from 'svelte';
	import Check from '@lucide/svelte/icons/check';
	import Copy from '@lucide/svelte/icons/copy';
	import Download from '@lucide/svelte/icons/download';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import List from '@lucide/svelte/icons/list';
	import Loader2 from '@lucide/svelte/icons/loader-2';
	import Monitor from '@lucide/svelte/icons/monitor';
	import Moon from '@lucide/svelte/icons/moon';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Sun from '@lucide/svelte/icons/sun';
	import { toast } from 'svelte-sonner';
	import { setMode, userPrefersMode } from 'mode-watcher';
	import * as api from '#lib/api';
	import { app } from '#lib/app.svelte';
	import { i18n, t } from '#lib/i18n.svelte';
	import { openExternal } from '#lib/wails';
	import type { UpdateInfo, UpdateProgress } from '#lib/types';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Progress } from '#lib/components/ui/progress/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';

	let baseUrl = $state(app.settings?.ai.baseUrl ?? '');
	let model = $state(app.settings?.ai.model ?? '');
	let apiKey = $state('');
	let keyInput = $state<HTMLInputElement | null>(null);
	let clearApiKey = $state(false);
	let saving = $state(false);
	let testing = $state(false);

	const configured = $derived(app.settings?.ai.configured ?? false);
	const hint = $derived(app.settings?.ai.apiKeyHint ?? '');

	$effect(() => {
		if (keyInput) keyInput.type = 'pass' + 'word';
	});

	let loadingModels = $state(false);
	let modelOptions = $state<string[]>([]);
	let modelError = $state('');
	let modelFilter = $state('');
	let autoFetchTimer: ReturnType<typeof setTimeout> | null = null;
	let lastModelFetchSignature = '';

	const filteredModels = $derived(
		modelOptions.filter((option) => option.toLowerCase().includes(modelFilter.trim().toLowerCase()))
	);

	let checkingUpdate = $state(false);
	let installingUpdate = $state(false);
	let updateInfo = $state<UpdateInfo | null>(null);
	let updateError = $state('');
	let restarting = $state(false);
	let updateProgress = $state<UpdateProgress>({
		active: false,
		stage: '',
		downloaded: 0,
		total: 0,
		percent: 0
	});
	let progressTimer: ReturnType<typeof setInterval> | null = null;

	const updateSource = $derived(updateInfo?.source === 'gh-proxy' ? 'gh-proxy' : 'GitHub');
	const progressPercent = $derived(
		updateProgress.total > 0 ? Math.min(100, Math.round(updateProgress.percent)) : 0
	);
	const progressLabel = $derived(
		updateProgress.stage === 'downloading'
			? t('正在下载…')
			: updateProgress.stage === 'installing'
				? t('正在安装…')
				: updateProgress.stage === 'done'
					? t('正在重启…')
					: t('准备中…')
	);

	onDestroy(() => {
		stopProgressPolling();
		stopAutoFetch();
	});

	function stopProgressPolling(): void {
		if (progressTimer) {
			clearInterval(progressTimer);
			progressTimer = null;
		}
	}

	function stopAutoFetch(): void {
		if (autoFetchTimer) {
			clearTimeout(autoFetchTimer);
			autoFetchTimer = null;
		}
	}

	function resetModelOptions(): void {
		modelOptions = [];
		modelFilter = '';
		modelError = '';
		lastModelFetchSignature = '';
	}

	function handleBaseUrlInput(): void {
		resetModelOptions();
	}

	function handleApiKeyInput(): void {
		clearApiKey = false;
		resetModelOptions();
		stopAutoFetch();
		if (!baseUrl.trim() || !apiKey.trim()) return;
		autoFetchTimer = setTimeout(() => {
			autoFetchTimer = null;
			autoFetchModels();
		}, 800);
	}

	function handleAutoFetchBlur(): void {
		stopAutoFetch();
		autoFetchModels();
	}

	function autoFetchModels(): void {
		const key = apiKey.trim();
		const endpoint = baseUrl.trim();
		if (!key || !endpoint || loadingModels) return;
		const signature = endpoint + '|' + key;
		if (signature === lastModelFetchSignature) return;
		lastModelFetchSignature = signature;
		void fetchModels();
	}

	function startProgressPolling(): void {
		stopProgressPolling();
		updateProgress = { active: true, stage: 'preparing', downloaded: 0, total: 0, percent: 0 };
		progressTimer = setInterval(async () => {
			try {
				updateProgress = await api.updateStatus();
			} catch {
				// The process may be restarting.
			}
		}, 300);
	}

	function formatBytes(value: number): string {
		const bytes = Number(value || 0);
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	async function checkForUpdate(): Promise<void> {
		if (checkingUpdate || installingUpdate) return;
		checkingUpdate = true;
		updateError = '';
		updateInfo = null;
		try {
			updateInfo = await api.checkUpdate();
		} catch (error) {
			updateError = error instanceof Error ? error.message : String(error);
		} finally {
			checkingUpdate = false;
		}
	}

	async function installUpdate(): Promise<void> {
		if (!updateInfo?.available || installingUpdate) return;
		installingUpdate = true;
		updateError = '';
		startProgressPolling();
		try {
			await api.applyUpdate();
			restarting = true;
			toast.success(t('更新已安装，正在重启…'));
			await waitForRestart();
		} catch (error) {
			updateError = error instanceof Error ? error.message : String(error);
			restarting = false;
		} finally {
			stopProgressPolling();
			installingUpdate = false;
		}
	}

	async function waitForRestart(): Promise<void> {
		await new Promise((resolve) => setTimeout(resolve, 1500));
		for (let attempt = 0; attempt < 60; attempt++) {
			try {
				await api.health();
				location.reload();
				return;
			} catch {
				await new Promise((resolve) => setTimeout(resolve, 1000));
			}
		}
		restarting = false;
		updateError = t('重启超时，请手动重新打开 WeekTodo。');
	}

	async function save(): Promise<void> {
		saving = true;
		const ok = await app.saveAISettings({ baseUrl, model, apiKey, clearApiKey });
		saving = false;
		if (ok) {
			apiKey = '';
			clearApiKey = false;
		}
	}

	async function test(): Promise<void> {
		testing = true;
		await app.testAISettings({ baseUrl, model, apiKey });
		testing = false;
	}

	async function fetchModels(): Promise<void> {
		stopAutoFetch();
		if (loadingModels) return;
		modelError = '';
		if (!baseUrl.trim()) {
			modelError = t('请先填写 Base URL');
			return;
		}
		if (!apiKey.trim() && !configured) {
			modelError = t('请先填写 API Key');
			return;
		}
		lastModelFetchSignature = baseUrl.trim() + '|' + apiKey.trim();
		loadingModels = true;
		try {
			const result = await api.fetchAIModels({ baseUrl, apiKey });
			modelOptions = result.models;
			modelFilter = '';
			if (result.models.length === 0) {
				modelError = t('接口没有返回模型列表');
			}
		} catch (error) {
			modelOptions = [];
			modelError = error instanceof Error ? error.message : String(error);
		} finally {
			loadingModels = false;
		}
	}

	function selectModel(option: string): void {
		model = option;
		modelOptions = [];
		modelFilter = '';
		modelError = '';
	}

	async function copyPath(): Promise<void> {
		if (!app.meta) return;
		try {
			await navigator.clipboard.writeText(app.meta.dataDir);
			toast.success(t('路径已复制'));
		} catch {
			toast.error(t('复制失败'));
		}
	}
</script>

<Dialog.Root bind:open={app.settingsOpen}>
	<Dialog.Content class="max-h-[88vh] overflow-y-auto sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{t('设置')}</Dialog.Title>
			<Dialog.Description>{t('配置 AI 周报润色和界面外观。')}</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-6 py-1">
			<section class="space-y-3">
				<div class="flex items-center justify-between">
					<div>
						<h3 class="text-sm font-medium">{t('AI 周报润色')}</h3>
						<p class="mt-0.5 text-xs text-muted-foreground">
							{t('兼容 OpenAI 的 /chat/completions 接口。')}
						</p>
					</div>
					{#if configured}
						<span class="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
							<Check class="size-3.5" /> {t('已配置')}
						</span>
					{/if}
				</div>

				<div class="space-y-1.5">
					<Label for="ai-base-url">Base URL</Label>
					<Input
						id="ai-base-url"
						bind:value={baseUrl}
						oninput={handleBaseUrlInput}
						onblur={handleAutoFetchBlur}
						placeholder="https://api.deepseek.com/v1"
						autocomplete="off"
					/>
				</div>
				<div class="space-y-1.5">
					<Label for="ai-key">API Key</Label>
					<Input
						id="ai-key"
						bind:ref={keyInput}
						bind:value={apiKey}
						oninput={handleApiKeyInput}
						onblur={handleAutoFetchBlur}
						placeholder={configured ? t('已保存 {hint}，留空保持不变', { hint }) : 'sk-...'}
						autocomplete="off"
					/>
					<p class="text-xs text-muted-foreground">
						{t('Key 只保存在本机 settings.json（权限 0600），不会回传前端。')}
					</p>
				</div>

				<div class="space-y-1.5">
					<Label for="ai-model">{t('模型')}</Label>
					<div class="flex gap-2">
						<Input
							id="ai-model"
							bind:value={model}
							placeholder="deepseek-chat"
							autocomplete="off"
							class="min-w-0 flex-1"
						/>
						<Button
							variant="outline"
							size="sm"
							class="shrink-0"
							onclick={fetchModels}
							disabled={loadingModels}
						>
							{#if loadingModels}
								<Loader2 class="animate-spin" /> {t('获取中…')}
							{:else}
								<List /> {t('获取列表')}
							{/if}
						</Button>
					</div>
					{#if modelError}
						<p class="text-xs text-destructive">{modelError}</p>
					{/if}
					{#if modelOptions.length > 0}
						<div class="overflow-hidden rounded-lg border">
							<div class="border-b p-1.5">
								<Input
									bind:value={modelFilter}
									placeholder={t('筛选 {count} 个模型…', { count: modelOptions.length })}
									class="h-7 text-xs"
								/>
							</div>
							<div class="max-h-44 overflow-y-auto p-1">
								{#each filteredModels as option (option)}
									<button
										type="button"
										class="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors hover:bg-muted"
										onclick={() => selectModel(option)}
									>
										<span class="truncate">{option}</span>
										{#if model === option}
											<Check class="size-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
										{/if}
									</button>
								{:else}
									<p class="px-2 py-3 text-xs text-muted-foreground">{t('没有匹配的模型')}</p>
								{/each}
							</div>
						</div>
					{/if}
				</div>

				<div class="flex flex-wrap items-center gap-2">
					<Button variant="outline" size="sm" onclick={test} disabled={testing || saving}>
						{#if testing}<Loader2 class="animate-spin" /> {t('测试中…')}
						{:else}{t('测试连接')}{/if}
					</Button>
					{#if configured}
						<Button
							variant="ghost"
							size="sm"
							class="text-muted-foreground"
							onclick={() => {
								clearApiKey = true;
								apiKey = '';
							}}
						>
							{t('清除已保存的 Key')}
						</Button>
					{/if}
					{#if clearApiKey}
						<span class="text-xs text-destructive">{t('保存后将清除 Key')}</span>
					{/if}
				</div>
			</section>

			<Separator />

			<section class="space-y-2">
				<h3 class="text-sm font-medium">{t('外观')}</h3>
				<div class="flex gap-1.5">
					<Button
						variant={userPrefersMode.current === 'light' ? 'secondary' : 'outline'}
						size="sm"
						onclick={() => setMode('light')}
					>
						<Sun /> {t('浅色')}
					</Button>
					<Button
						variant={userPrefersMode.current === 'dark' ? 'secondary' : 'outline'}
						size="sm"
						onclick={() => setMode('dark')}
					>
						<Moon /> {t('深色')}
					</Button>
					<Button
						variant={userPrefersMode.current === undefined ? 'secondary' : 'outline'}
						size="sm"
						onclick={() => setMode('system')}
					>
						<Monitor /> {t('跟随系统')}
					</Button>
				</div>
			</section>

			<Separator />

			<section class="space-y-2">
				<h3 class="text-sm font-medium">{t('语言')}</h3>
				<div class="flex gap-1.5">
					<Button
						variant={i18n.language === 'zh' ? 'secondary' : 'outline'}
						size="sm"
						onclick={() => app.setLanguage('zh')}
					>
						{t('简体中文')}
					</Button>
					<Button
						variant={i18n.language === 'en' ? 'secondary' : 'outline'}
						size="sm"
						onclick={() => app.setLanguage('en')}
					>
						{t('English')}
					</Button>
				</div>
			</section>

			<Separator />

			<section class="space-y-2">
				<div class="flex items-start justify-between gap-3">
					<div>
						<h3 class="text-sm font-medium">{t('应用更新')}</h3>
						<p class="mt-0.5 text-xs text-muted-foreground">
							{t('当前版本 {version}', { version: app.meta?.version ?? 'dev' })}
						</p>
					</div>
					<Button
						variant="outline"
						size="sm"
						onclick={checkForUpdate}
						disabled={checkingUpdate || installingUpdate}
					>
						{#if checkingUpdate}
							<RefreshCw class="animate-spin" /> {t('检查中…')}
						{:else}
							<RefreshCw /> {t('检查更新')}
						{/if}
					</Button>
				</div>

				{#if updateError}
					<p class="text-xs text-destructive">{updateError}</p>
				{/if}

				{#if updateInfo}
					{#if updateInfo.available}
						<div class="space-y-2 rounded-lg border p-3">
							<div class="flex flex-wrap items-center gap-2">
								<span class="text-sm font-medium">{t('发现 v{version}', { version: updateInfo.latest })}</span>
								<Badge variant="secondary">{t('来源 {source}', { source: updateSource })}</Badge>
								{#if updateInfo.published_at}
									<span class="text-xs text-muted-foreground">
										{updateInfo.published_at.slice(0, 10)}
									</span>
								{/if}
							</div>
							{#if updateInfo.notes}
								<pre
									class="max-h-40 overflow-auto whitespace-pre-wrap rounded-md bg-muted p-2 text-xs text-muted-foreground">{updateInfo.notes}</pre>
							{/if}
							<div class="flex flex-wrap items-center gap-2">
								<Button size="sm" onclick={installUpdate} disabled={installingUpdate}>
									{#if installingUpdate}
										<Loader2 class="animate-spin" /> {t('正在更新…')}
									{:else}
										<Download /> {t('安装 v{version}', { version: updateInfo.latest })}
									{/if}
								</Button>
								{#if updateInfo.release_url}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openExternal(updateInfo!.release_url)}
									>
										<ExternalLink /> {t('查看 Release')}
									</Button>
								{/if}
							</div>
							{#if installingUpdate || restarting}
								<Progress value={progressPercent} />
								<p class="text-xs text-muted-foreground">
									{progressLabel}
									{#if updateProgress.total > 0}
										· {progressPercent}% ({formatBytes(updateProgress.downloaded)} /
										{formatBytes(updateProgress.total)})
									{/if}
								</p>
							{/if}
						</div>
					{:else}
						<p class="text-xs text-muted-foreground">
							{updateInfo.dev ? t('当前是开发版本，无法检查更新。') : t('当前已是最新版本。')}
						</p>
					{/if}
				{/if}
			</section>

			<Separator />

			<section class="space-y-2">
				<h3 class="text-sm font-medium">{t('数据')}</h3>
				<div class="flex items-center gap-2">
					<code class="min-w-0 flex-1 truncate rounded-md bg-muted px-2 py-1.5 text-xs">
						{app.meta?.dataDir ?? '—'}
					</code>
					<Button variant="ghost" size="icon-sm" onclick={copyPath} title={t('复制路径')}>
						<Copy />
					</Button>
				</div>
				<p class="text-xs text-muted-foreground">
					WeekTodo {app.meta?.version ?? ''}
					{#if app.meta?.commit && app.meta.commit !== 'unknown'}
						· {app.meta.commit.slice(0, 7)}
					{/if}
					· {t('数据以 JSON 保存，每天首次保存时会自动备份，保留 14 天。')}
				</p>
			</section>
		</div>

		<Dialog.Footer>
			<Button variant="outline" size="sm" onclick={() => (app.settingsOpen = false)}>{t('关闭')}</Button>
			<Button size="sm" onclick={save} disabled={saving}>
				{#if saving}<Loader2 class="animate-spin" /> {t('保存中…')}
				{:else}{t('保存设置')}{/if}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
