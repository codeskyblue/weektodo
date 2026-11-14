type WailsRuntime = {
	EventsEmit?: (event: string, ...data: unknown[]) => void;
	EventsOn?: (event: string, callback: (...data: unknown[]) => void) => () => void;
	BrowserOpenURL?: (url: string) => void;
};

type WailsBrowser = {
	OpenURL?: (url: string) => void;
};

type WailsRuntimeV3 = {
	Browser?: WailsBrowser;
};

type WailsWindow = Window & {
	runtime?: WailsRuntime;
	wails?: WailsRuntimeV3;
	_wails?: { invoke?: (message: string) => void };
};

function runtime(): WailsRuntime | null {
	if (typeof window === 'undefined') return null;
	const value = (window as WailsWindow).runtime;
	return value ?? null;
}

function desktopRuntime(): WailsRuntimeV3 | null {
	if (typeof window === 'undefined') return null;
	const value = (window as WailsWindow).wails;
	return value?.Browser?.OpenURL ? value : null;
}

function isDesktop(): boolean {
	if (typeof window === 'undefined') return false;
	return typeof (window as WailsWindow)._wails?.invoke === 'function';
}

// Wails v3 dropped the v2 `window.runtime` global. Its runtime is an ES module
// that has to be imported explicitly; it registers `window.wails`.
const RUNTIME_MODULE = '/wails/runtime.js';
let runtimePromise: Promise<WailsRuntimeV3 | null> | null = null;

function loadDesktopRuntime(): Promise<WailsRuntimeV3 | null> {
	const loaded = desktopRuntime();
	if (loaded) return Promise.resolve(loaded);
	return (runtimePromise ??= import(/* @vite-ignore */ RUNTIME_MODULE)
		.then((module: WailsRuntimeV3) => (module?.Browser?.OpenURL ? module : desktopRuntime()))
		.catch(() => null));
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

export function openExternal(url: string): void {
	if (!url) return;
	const desktop = desktopRuntime();
	if (desktop?.Browser?.OpenURL) {
		desktop.Browser.OpenURL(url);
		return;
	}
	if (isDesktop()) {
		// WKWebView silently drops `window.open`, so use the native bridge.
		void loadDesktopRuntime().then((rt) => rt?.Browser?.OpenURL?.(url));
		return;
	}
	window.open(url, '_blank', 'noopener,noreferrer');
}
