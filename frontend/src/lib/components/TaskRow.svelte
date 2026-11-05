<script lang="ts">
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import ArrowDownToLine from '@lucide/svelte/icons/arrow-down-to-line';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import Ban from '@lucide/svelte/icons/ban';
	import MoreHorizontal from '@lucide/svelte/icons/more-horizontal';
	import Plus from '@lucide/svelte/icons/plus';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import { app } from '#lib/app.svelte';
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

	const task = $derived(row.task);
	const done = $derived(task.status === 'done');
	const dropped = $derived(task.status === 'dropped');
	const indeterminate = $derived(isIndeterminate(task));

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
	class="group/row relative flex items-start gap-2 rounded-lg py-1 pr-2 transition-colors hover:bg-muted/55 focus-within:bg-muted/40"
	style={`padding-left: ${0.5 + row.depth * 1.15}rem`}
	data-depth={row.depth}
>
	<div class="pt-1.5">
		<Checkbox
			checked={done}
			{indeterminate}
			disabled={dropped}
			onCheckedChange={() => app.toggleTask(task.id)}
			aria-label={done ? `取消完成 ${task.text || '任务'}` : `完成 ${task.text || '任务'}`}
		/>
	</div>

	<div class="min-w-0 flex-1">
		<div class="flex min-h-7 items-center gap-1">
			{#if task.carriedFrom}
				<ArrowDownToLine
					class="size-3.5 shrink-0 text-muted-foreground/70"
					aria-label="从上一周带入"
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
				placeholder="写点什么…"
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
					已废弃
				</span>
			{/if}
			{#if task.children.length > 0}
				<span class="hidden shrink-0 text-[10px] tabular-nums text-muted-foreground/50 sm:inline">
					{task.children.length}
				</span>
			{/if}
			<div
				class="flex shrink-0 items-center gap-0.5 {task.day
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
								title="任务操作"
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
							<Plus class="size-4" /> 添加子任务
						</DropdownMenu.Item>
						<DropdownMenu.Item onSelect={() => app.move(task.id, -1)}>
							<ArrowUp class="size-4" /> 上移
						</DropdownMenu.Item>
						<DropdownMenu.Item onSelect={() => app.move(task.id, 1)}>
							<ArrowDown class="size-4" /> 下移
						</DropdownMenu.Item>
						<DropdownMenu.Separator />
						{#if dropped}
							<DropdownMenu.Item onSelect={() => app.restoreTask(task.id)}>
								<RotateCcw class="size-4" /> 恢复任务
							</DropdownMenu.Item>
						{:else}
							<DropdownMenu.Item onSelect={() => app.dropTask(task.id)}>
								<Ban class="size-4" /> 废弃任务
							</DropdownMenu.Item>
						{/if}
						<DropdownMenu.Item
							variant="destructive"
							onSelect={() => app.deleteTask(task.id)}
						>
							<Trash2 class="size-4" /> 删除任务
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>
		</div>
	</div>
</div>
