//go:build windows

package index

// Windows does not permit FlushFileBuffers on a directory handle opened by
// os.Open. The file itself has already been flushed and closed before its
// atomic rename, so there is no supported directory fsync left to perform.
func syncDirectory(string) error {
	return nil
}
