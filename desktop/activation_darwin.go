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
*/
import "C"

func setAppRegular() {
	C.weektodoSetActivationPolicy(0)
}

func setAppAccessory() {
	C.weektodoSetActivationPolicy(1)
}
