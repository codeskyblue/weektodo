type WailsRuntime = {
	EventsEmit?: (event: string, ...data: unknown[]) => void;
	EventsOn?: (event: string, callback: (...data: unknown[]) => void) => () => void;
	BrowserOpenURL?: (url: string) => void;
};

function runtime(): WailsRuntime | null {
	if (typeof window === 'undefined') return null;
	const value = (window as unknown as { runtime?: WailsRuntime }).runtime;
	return value ?? null;
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
	try {
		const rt = runtime();
		if (rt?.BrowserOpenURL) {
			rt.BrowserOpenURL(url);
			return;
		}
	} catch {
		// Browser mode falls through.
	}
	window.open(url, '_blank', 'noopener,noreferrer');
}
