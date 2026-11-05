<script lang="ts">
	import CalendarPlus from '@lucide/svelte/icons/calendar-plus';
	import Inbox from '@lucide/svelte/icons/inbox';
	import ListTodo from '@lucide/svelte/icons/list-todo';
	import { app } from '#lib/app.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Card, CardContent } from '#lib/components/ui/card/index.js';

	let { filtered = false }: { filtered?: boolean } = $props();
</script>

{#if filtered}
	<div class="flex flex-col items-center gap-3 rounded-xl border border-dashed px-6 py-14 text-center">
		<div class="rounded-full bg-muted p-3 text-muted-foreground">
			<Inbox class="size-5" />
		</div>
		<div>
			<div class="text-sm font-medium">没有符合筛选的任务</div>
			<div class="mt-1 text-xs text-muted-foreground">换个筛选条件看看。</div>
		</div>
		<Button variant="outline" size="sm" onclick={() => (app.filter = 'all')}>查看全部</Button>
	</div>
{:else}
	<Card class="border-dashed shadow-none">
		<CardContent class="flex flex-col items-center gap-4 px-6 py-14 text-center">
			<div class="rounded-full bg-muted p-3 text-muted-foreground">
				<ListTodo class="size-5" />
			</div>
			<div>
				<div class="text-sm font-medium">这一周还没有任务</div>
				<div class="mt-1 text-xs text-muted-foreground">
					从一条小任务开始，或者把上周没做完的带过来。
				</div>
			</div>
			<div class="flex flex-wrap items-center justify-center gap-2">
				<Button size="sm" onclick={() => app.addTask()}>
					<CalendarPlus /> 添加第一条
				</Button>
				<Button
					variant="outline"
					size="sm"
					disabled={app.carryBusy}
					onclick={() => app.openCarryOver()}
				>
					<Inbox /> 带入上周未完成
				</Button>
			</div>
		</CardContent>
	</Card>
{/if}
