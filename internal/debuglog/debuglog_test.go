package debuglog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenWritesToStateDirAndRedacts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	l, closeFn, err := Open(false)
	if err != nil {
		t.Fatal(err)
	}
	l.Info("loaded", "title", l.Redact("Dentist"))
	if err := closeFn(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "sundial", "debug.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "Dentist") || !strings.Contains(string(b), "[redacted]") {
		t.Errorf("log not redacted: %s", b)
	}
	v, closeV, _ := Open(true)
	if v.Redact("Dentist") != "Dentist" {
		t.Error("verbose must not redact")
	}
	_ = closeV()
}
