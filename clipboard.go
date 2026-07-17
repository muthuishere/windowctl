package windowctl

// GetClipboard returns the system clipboard's current text. Non-string
// clipboard contents (an image, a file) read back as an empty string,
// not an error.
func GetClipboard() (string, error) {
	return defaultAdapter.Clipboard()
}

// SetClipboard replaces the system clipboard with text. Paired with a
// paste chord (`key cmd+v`) this is the reliable, layout- and
// IME-independent way to insert text into a focused field — synthetic
// per-character typing is subject to keycode/layout mismatch and
// first-responder drops; the clipboard is not.
func SetClipboard(text string) error {
	return defaultAdapter.SetClipboard(text)
}
