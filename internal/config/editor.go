package config

import "os/exec"

// EditorCommand builds the command that opens path in $EDITOR (default vi).
// Running through `sh -c` lets EDITOR carry flags (e.g. "code -w") while
// the path is passed safely as $1.
func EditorCommand(getenv func(string) string, path string) *exec.Cmd {
	editor := getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
}
