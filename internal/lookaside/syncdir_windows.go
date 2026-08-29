//go:build windows

package lookaside

// Windows does not support syncing a directory handle opened through os.Open.
// Published files are flushed before their atomic link becomes visible.
func syncDirectory(string) error {
	return nil
}
