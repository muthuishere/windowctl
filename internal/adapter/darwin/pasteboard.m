// pasteboard.m — NSPasteboard read/write for the darwin adapter.
#import <AppKit/AppKit.h>
#include "pasteboard.h"
#include <string.h>

int wctl_clipboard_get(char *buf, int cap) {
    if (cap <= 0) return -1;
  @autoreleasepool {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    NSString *s = [pb stringForType:NSPasteboardTypeString];
    if (s == nil) { buf[0] = '\0'; return -1; }
    const char *utf8 = [s UTF8String];
    if (utf8 == NULL) { buf[0] = '\0'; return -1; }
    int len = (int)strlen(utf8);
    int n = len < cap - 1 ? len : cap - 1;
    memcpy(buf, utf8, n);
    buf[n] = '\0';
    return len;
  }
}

int wctl_clipboard_set(const char *utf8) {
  @autoreleasepool {
    NSString *s = [NSString stringWithUTF8String:(utf8 ? utf8 : "")];
    if (s == nil) return -5;
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    [pb clearContents];
    BOOL ok = [pb setString:s forType:NSPasteboardTypeString];
    return ok ? 0 : -5;
  }
}
