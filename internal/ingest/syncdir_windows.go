//go:build windows

package ingest

// Windows does not support syncing a directory handle opened through os.Open.
// Every staged file is flushed before its atomic rename or link is exposed.
func syncDirectory(string) error {
	return nil
}
