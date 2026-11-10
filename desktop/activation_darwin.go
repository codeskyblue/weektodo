//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void weektodoSetActivationPolicy(int policy) {
	if ([NSThread isMainThread]) {
		[NSApp setActivationPolicy:policy];
	} else {
		dispatch_sync(dispatch_get_main_queue(), ^{
			[NSApp setActivationPolicy:policy];
		});
	}
}

static void weektodoActivate() {
	if ([NSThread isMainThread]) {
		[NSApp activateIgnoringOtherApps:YES];
	} else {
		dispatch_sync(dispatch_get_main_queue(), ^{
			[NSApp activateIgnoringOtherApps:YES];
		});
	}
}
*/
import "C"

func setAppRegular() {
	C.weektodoSetActivationPolicy(0)
}

func setAppAccessory() {
	C.weektodoSetActivationPolicy(1)
}

// activateApp brings the app to the front. Switching back to Regular only adds
// the app to the Dock and Cmd+Tab list once it is activated; showing the window
// alone leaves the app behind the current foreground app.
func activateApp() {
	C.weektodoActivate()
}
