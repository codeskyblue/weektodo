<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import { t } from '#lib/i18n.svelte';
	import { addTaskFromBlankArea } from '#lib/task-add';
	import type { TaskRow } from '#lib/types';

	let { weekId, rows }: { weekId: string; rows: TaskRow[] } = $props();
</script>

<button
	type="button"
	class="group/blank flex min-h-10 w-full flex-1 cursor-text items-start px-2 pt-1 text-left outline-none transition-colors focus-visible:bg-muted/30"
	aria-label={t('新建任务')}
	onclick={(event) => addTaskFromBlankArea(event, weekId, rows)}
>
	<span
		class="pointer-events-none flex h-8 items-center text-muted-foreground/25 opacity-0 transition-opacity group-hover/blank:opacity-100 max-sm:opacity-50"
		aria-hidden="true"
	>
		<Plus class="size-3.5" />
	</span>
</button>
