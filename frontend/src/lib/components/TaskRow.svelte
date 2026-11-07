<script lang="ts">
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import ArrowDownToLine from '@lucide/svelte/icons/arrow-down-to-line';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import GripVertical from '@lucide/svelte/icons/grip-vertical';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import Ban from '@lucide/svelte/icons/ban';
	import MoreHorizontal from '@lucide/svelte/icons/more-horizontal';
	import Plus from '@lucide/svelte/icons/plus';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { app } from '#lib/app.svelte';
	import { t } from '#lib/i18n.svelte';
	import { isIndeterminate } from '#lib/tree';
	import type { TaskRow } from '#lib/types';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import DayPicker from './DayPicker.svelte';

	let {
		row,
		focusRequest
	}: {
		row: TaskRow;
		focusRequest: { id: string; caret: 'start' | 'end'; nonce: number } | null;
	} = $props();

	let inputEl = $state<HTMLInputElement | null>(null);
	let rowEl = $state<HTMLElement | null>(null);

	const task = $derived(row.task);
	const done = $derived(task.status === 'done');
	const dropped = $derived(task.status === 'dropped');
	const indeterminate = $derived(isIndeterminate(task));
	const moveOptions = $derived(app.moveOptions(task.id));

	$effect(() => {
		const request = focusRequest;
		if (!request || request.id !== task.id || !inputEl) return;
		inputEl.focus();
		const position = request.caret === 'start' ? 0 : inputEl.value.length;
		inputEl.setSelectionRange(position, position);
	});

	function handleKeydown(event: KeyboardEvent): void {
		const input = event.currentTarget as HTMLInputElement;
		const atStart = input.selectionStart === 0 && input.selectionEnd === 0;
		if (event.key === 'Enter') {
			event.preventDefault();
			if (!task.text.trim()) {
				// Enter on an empty nested task behaves like Shift+Tab,
				// so users can exit a sub-list without leaving the keyboard.
				if (row.parent) app.outdent(task.id);
				else input.blur();
				return;
			}
			app.addSibling(task.id);
			return;
		}
		if (event.key === 'Tab') {
			event.preventDefault();
			if (!atStart) return;
			if (event.shiftKey) app.outdent(task.id);
			else app.indent(task.id);
			return;
		}
		if (
			event.key === 'Backspace' &&
			atStart &&
			task.text.length === 0 &&
			task.children.length === 0
		) {
			event.preventDefault();
			app.removeEmptyTask(task.id);
			return;
		}
		if (event.altKey && event.key === 'ArrowUp') {
			event.preventDefault();
			app.move(task.id, -1);
			return;
		}
		if (event.altKey && event.key === 'ArrowDown') {
			event.preventDefault();
			app.move(task.id, 1);
			return;
		}
		if (event.key === 'Escape') {
			input.blur();
		}
	}

	function handleDragStart(event: DragEvent): void {
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = 'move';
			event.dataTransfer.setData('text/plain', task.id);
		}
		if (event.dataTransfer && rowEl) {
			const rect = rowEl.getBoundingClientRect();
			const ghost = rowEl.cloneNode(true) as HTMLElement;
			ghost.style.position = 'fixed';
			ghost.style.top = '-1000px';
			ghost.style.left = '-1000px';
			ghost.style.width = `${Math.min(rect.width, 480)}px`;
			ghost.style.opacity = '0.7';
			ghost.style.pointerEvents = 'none';
			ghost.style.overflow = 'hidden';
			ghost.style.zIndex = '9999';
			ghost.style.background = 'var(--background, #fff)';
			ghost.style.borderRadius = '0.5rem';
			ghost.style.boxShadow = '0 10px 30px rgb(0 0 0 / 0.18)';
			const sourceInputs = rowEl.querySelectorAll('input');
			const ghostInputs = ghost.querySelectorAll('input');
			sourceInputs.forEach((source, index) => {
				const target = ghostInputs[index] as HTMLInputElement | undefined;
				if (!target) return;
				if (source instanceof HTMLInputElement) {
					target.value = source.value;
					if (source.type === 'checkbox') target.checked = source.checked;
				}
			});
			document.body.appendChild(ghost);
			const offsetX = Math.max(0, Math.min(event.clientX - rect.left, ghost.offsetWidth - 1));
			const offsetY = Math.max(0, Math.min(event.clientY - rect.top, ghost.offsetHeight - 1));
			event.dataTransfer.setDragImage(ghost, offsetX, offsetY);
			requestAnimationFrame(() => ghost.remove());
		}
		app.startTaskDrag(task.id);
	}

	function handleBlur(event: FocusEvent): void {
		const input = event.currentTarget as HTMLInputElement;
		const value = input.value;
		if (value.trim() === '' && task.children.length === 0) {
			app.removeEmptyTask(task.id);
			return;
		}
		if (value !== value.trim()) {
			input.value = value.trim();
			app.updateText(task.id, value.trim());
		}
	}
</script>

<div
	bind:this={rowEl}
	class="group/row relative flex items-start gap-1 rounded-lg py-1 pr-2 transition-colors hover:bg-muted/55 focus-within:bg-muted/40 {app.dropTaskId ===
	task.id
		? 'bg-primary/5 ring-2 ring-primary/50 ring-inset'
		: ''} {app.draggingTaskId === task.id ? 'opacity-50' : ''}"
	style={`padding-left: ${0.5 + row.depth * 1.15}rem`}
	data-depth={row.depth}
	role="listitem"
	ondragover={(event) => {
		if (!app.draggingTaskId) return;
		event.stopPropagation();
		if (!app.canDropOnTask(task.id)) return;
		event.preventDefault();
		if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
		app.setTaskDropTarget(task.id);
	}}
	ondragleave={(event) => {
		const related = event.relatedTarget as Node | null;
		if (related && event.currentTarget.contains(related)) return;
		app.clearTaskDropTarget(task.id);
	}}
	ondrop={(event) => {
		event.preventDefault();
		event.stopPropagation();
		void app.dropOnTask(task.id);
	}}
>
	<div class="flex shrink-0 items-center pt-1.5">
		<button
			type="button"
			draggable="true"
			class="cursor-grab rounded p-0.5 text-muted-foreground/40 opacity-0 transition-opacity hover:text-muted-foreground group-hover/row:opacity-100 group-focus-within/row:opacity-100 active:cursor-grabbing"
			title={t('拖动任务')}
			aria-label={t('拖动任务')}
			ondragstart={handleDragStart}
			ondragend={() => app.endTaskDrag()}
			onclick={(event) => event.preventDefault()}
		>
			<GripVertical class="size-4" />
		</button>
	</div>
	<div class="pt-1.5">
		<Checkbox
			checked={done}
			{indeterminate}
			disabled={dropped}
			onCheckedChange={() => app.toggleTask(task.id)}
			aria-label={done ? t('取消完成 {title}', { title: task.text || t('任务') }) : t('完成 {title}', { title: task.text || t('任务') })}
		/>
	</div>

	<div class="min-w-0 flex-1">
		<div class="flex min-h-7 items-center gap-1">
			{#if task.carriedFrom}
				<ArrowDownToLine
					class="size-3.5 shrink-0 text-muted-foreground/70"
					aria-label={t('从上一周带入')}
				/>
			{/if}
			<input
				bind:this={inputEl}
				value={task.text}
				oninput={(event) => app.updateText(task.id, event.currentTarget.value)}
				onkeydown={handleKeydown}
				onblur={handleBlur}
				maxlength="500"
				autocomplete="off"
				spellcheck="false"
				placeholder={t('写点什么…')}
				class="h-7 min-w-0 flex-1 rounded-md border-0 bg-transparent px-1 text-sm outline-none placeholder:text-muted-foreground/50 focus:bg-background/70 focus:ring-0 {done
					? 'text-muted-foreground line-through decoration-muted-foreground/50'
					: ''} {dropped
					? 'text-muted-foreground/60 line-through decoration-muted-foreground/40'
					: ''} {row.context ? 'text-muted-foreground' : ''}"
			/>
			{#if dropped}
				<span
					class="shrink-0 rounded-full border border-destructive/20 bg-destructive/10 px-1.5 py-0.5 text-[10px] leading-none text-destructive"
				>
					{t('已废弃')}
				</span>
			{/if}
			{#if task.children.length > 0}
				<span class="hidden shrink-0 text-[10px] tabular-nums text-muted-foreground/50 sm:inline">
					{task.children.length}
				</span>
			{/if}
			<div
				class="flex shrink-0 items-center gap-0.5 max-sm:opacity-100 {task.day
					? ''
					: 'opacity-0 group-hover/row:opacity-100 group-focus-within/row:opacity-100'}"
			>
				<DayPicker {task} />
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button
								{...props}
								variant="ghost"
								size="icon-xs"
								title={t('任务操作')}
								class="text-muted-foreground"
							>
								<MoreHorizontal />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end" class="w-44">
						<DropdownMenu.Item
							disabled={dropped}
							onSelect={() => app.addChild(task.id)}
						>
							<Plus class="size-4" /> {t('添加子任务')}
						</DropdownMenu.Item>
						<DropdownMenu.Item onSelect={() => app.move(task.id, -1)}>
							<ArrowUp class="size-4" /> {t('上移')}
						</DropdownMenu.Item>
						<DropdownMenu.Item onSelect={() => app.move(task.id, 1)}>
							<ArrowDown class="size-4" /> {t('下移')}
						</DropdownMenu.Item>
						{#if moveOptions.length > 0}
							<DropdownMenu.Separator />
							{#each moveOptions as option (option.week.id)}
								<DropdownMenu.Item
									onSelect={() => app.moveTaskToWeek(task.id, option.week.id)}
								>
									{#if option.direction === 'previous'}
										<ArrowLeft class="size-4" />
									{:else}
										<ArrowRight class="size-4" />
									{/if}
									{option.direction === 'previous' ? t('移到上一周') : t('移到下一周')}
								</DropdownMenu.Item>
							{/each}
						{/if}
						<DropdownMenu.Separator />
						{#if dropped}
							<DropdownMenu.Item onSelect={() => app.restoreTask(task.id)}>
								<RotateCcw class="size-4" /> {t('恢复任务')}
							</DropdownMenu.Item>
						{:else}
							<DropdownMenu.Item onSelect={() => app.dropTask(task.id)}>
								<Ban class="size-4" /> {t('废弃任务')}
							</DropdownMenu.Item>
						{/if}
						<DropdownMenu.Item
							variant="destructive"
							onSelect={() => app.deleteTask(task.id)}
						>
							<Trash2 class="size-4" /> {t('删除任务')}
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>
		</div>
	</div>
</div>
