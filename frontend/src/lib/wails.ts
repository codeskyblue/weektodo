type WailsRuntime = {
	EventsEmit?: (event: string, ...data: unknown[]) => void;
	EventsOn?: (event: string, callback: (...data: unknown[]) => void) => () => void;
};

type WailsDesktopRuntime = {
	Browser?: {
		OpenURL?: (url: string) => void | Promise<void>;
	};
};

type WailsDesktopWindow = Window & {
	wails?: WailsDesktopRuntime;
	_wails?: {
		invoke?: (message: string) => void;
	};
};

let desktopRuntimePromise: Promise<WailsDesktopRuntime | null> | null = null;

function runtime(): WailsRuntime | null {
	if (typeof window === 'undefined') return null;
	const value = (window as unknown as { runtime?: WailsRuntime }).runtime;
	return value ?? null;
}

function desktopWindow(): WailsDesktopWindow | null {
	if (typeof window === 'undefined') return null;
	return window as WailsDesktopWindow;
}

// In desktop mode Wails injects a minimal `window._wails` transport but not
// the full JS runtime. The runtime is served at /wails/runtime.js and must be
// imported before Browser.OpenURL is available. Browser mode has no transport,
// so skip the import there and let window.open handle the link.
function loadDesktopRuntime(): Promise<WailsDesktopRuntime | null> {
	const win = desktopWindow();
	const loaded = win?.wails;
	if (loaded?.Browser?.OpenURL) return Promise.resolve(loaded);
	if (!win?._wails?.invoke) return Promise.resolve(null);
	const runtimeURL = '/wails/runtime.js';
	desktopRuntimePromise ??= import(/* @vite-ignore */ runtimeURL)
		.then((module) => {
			const runtimeModule = module as WailsDesktopRuntime;
			win.wails = runtimeModule;
			return runtimeModule;
		})
		.catch(() => null);
	return desktopRuntimePromise;
}

export function emitWails(event: string, ...data: unknown[]): void {
	try {
		runtime()?.EventsEmit?.(event, ...data);
	} catch {
		// Browser mode has no Wails runtime.
	}
}

export function onWails(event: string, callback: (...data: unknown[]) => void): () => void {
	try {
		return runtime()?.EventsOn?.(event, callback) ?? (() => {});
	} catch {
		return () => {};
	}
}

function openWithDesktopRuntime(desktop: WailsDesktopRuntime, url: string): boolean {
	const openURL = desktop.Browser?.OpenURL;
	if (!openURL) return false;
	try {
		void Promise.resolve(openURL(url)).catch(() => {});
	} catch {
		return false;
	}
	return true;
}

export function openExternal(url: string): void {
	if (!url) return;
	const win = desktopWindow();
	if (win?.wails && openWithDesktopRuntime(win.wails, url)) return;
	if (win?._wails?.invoke) {
		void loadDesktopRuntime().then((desktop) => {
			if (desktop && openWithDesktopRuntime(desktop, url)) return;
			window.open(url, '_blank', 'noopener,noreferrer');
		});
		return;
	}
	window.open(url, '_blank', 'noopener,noreferrer');
}
