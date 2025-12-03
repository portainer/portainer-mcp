package tooldef

import (
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
)

func TestResolveToolsPath_DefaultToExeDir(t *testing.T) {
    got := ResolveToolsPath("", "tools.yaml")
    exe, err := os.Executable()
    if err != nil {
        t.Fatalf("executable path error: %v", err)
    }
    wantDir := filepath.Dir(exe)
    if filepath.Dir(got) != wantDir || filepath.Base(got) != "tools.yaml" {
        t.Fatalf("unexpected path: got %q, want dir %q and base tools.yaml", got, wantDir)
    }
}

func TestResolveToolsPath_EnvExpansion(t *testing.T) {
    tmp, err := os.MkdirTemp("", "mcp-tools-*")
    if err != nil {
        t.Fatalf("temp dir error: %v", err)
    }
    defer os.RemoveAll(tmp)

    var flagPath string
    var want string
    if runtime.GOOS == "windows" {
        os.Setenv("TEMP", tmp)
        flagPath = "%TEMP%\\tools.yaml"
        want = filepath.Join(tmp, "tools.yaml")
    } else {
        os.Setenv("TMPDIR", tmp)
        flagPath = "$TMPDIR/tools.yaml"
        want = filepath.Join(tmp, "tools.yaml")
    }

    got := ResolveToolsPath(flagPath, "tools.yaml")
    if got != want {
        t.Fatalf("unexpected expanded path: got %q, want %q", got, want)
    }
    if runtime.GOOS == "windows" && !strings.Contains(got, "\\") {
        t.Fatalf("windows path should contain backslashes: %q", got)
    }
}
