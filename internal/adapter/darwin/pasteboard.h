// pasteboard.h — C ABI for NSPasteboard read/write (implemented in
// pasteboard.m, compiled by cgo as Objective-C).
#ifndef WCTL_PASTEBOARD_H
#define WCTL_PASTEBOARD_H

// Writes up to cap-1 bytes of the pasteboard's string contents (UTF-8,
// NUL-terminated) into buf. Returns the number of bytes written (>=0),
// or -1 if the pasteboard holds no string. If the string is longer than
// cap-1 it is truncated and the full length is returned so the caller
// can retry with a bigger buffer.
int wctl_clipboard_get(char *buf, int cap);

// Replaces the pasteboard contents with the given UTF-8 string.
// Returns 0 on success, -5 on failure.
int wctl_clipboard_set(const char *utf8);

#endif
