package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	tmp := t.TempDir()
	root, err := filepath.EvalSymlinks(filepath.Join(tmp))
	if err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(root, "root")
	outside := filepath.Join(filepath.Dir(root), "outside")
	for _, dir := range []string{filepath.Join(root, "sub"), outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		rel  string
		want string
		ok   bool
	}{
		{"/", root, true},
		{"", root, true},
		{"/sub", filepath.Join(root, "sub"), true},
		{"sub/new.txt", filepath.Join(root, "sub", "new.txt"), true},
		{"/../outside", filepath.Join(root, "outside"), true}, // 词法清理后仍在根目录内
		{"/sub/../../outside", filepath.Join(root, "outside"), true},
		{"/escape", "", false},           // 软链指向根目录外
		{"/escape/a.txt", "", false},     // 软链下不存在的文件
		{"/escape/x/y/z.txt", "", false}, // 多级不存在
	}
	for _, c := range cases {
		got, err := resolvePath(root, c.rel)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("resolvePath(%q) = %q, %v; want %q", c.rel, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("resolvePath(%q) = %q; want error", c.rel, got)
		}
	}
}

func TestValidName(t *testing.T) {
	for _, name := range []string{"", " ", ".", "..", "a/b", `a\b`} {
		if _, err := validName(name); err == nil {
			t.Errorf("validName(%q) want error", name)
		}
	}
	if got, err := validName(" a.txt "); err != nil || got != "a.txt" {
		t.Errorf("validName trim got %q, %v", got, err)
	}
}
