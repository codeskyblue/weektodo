<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from '#lib/components/ui/sonner/index.js';

	let { children } = $props();
</script>

<ModeWatcher defaultMode="system" />
{@render children()}
<Toaster richColors position="bottom-right" closeButton />
