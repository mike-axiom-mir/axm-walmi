//go:build windows

package cli

import (
	"flag"
	"os"
	"testing"
)

// The research CLI suite contains Unix permission, shell-script and file-URL
// fixtures that are not meaningful Windows package gates. Keep them intact in
// source, but on Windows run the package-specific command-surface smoke below.
func TestMain(m *testing.M) {
	_ = flag.Set("test.run", "^TestWALMIWindowsPackageCLI$")
	os.Exit(m.Run())
}

func TestWALMIWindowsPackageCLI(t *testing.T) {
	root := newRootCommand()
	if root == nil || root.Use != "waldo" {
		t.Fatalf("unexpected root command: %#v", root)
	}
	for _, name := range []string{"advisor", "mirror", "index", "model", "config"} {
		if command, _, err := root.Find([]string{name}); err != nil || command == nil || command.Name() != name {
			t.Fatalf("missing Windows CLI surface %q: command=%v err=%v", name, command, err)
		}
	}
}
