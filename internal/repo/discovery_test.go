package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func makeRepo(t *testing.T, root, rel string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func makeWorktree(t *testing.T, root, rel string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Worktrees have .git as a file, not a directory.
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../main/.git/worktrees/wt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()

	makeRepo(t, root, "alpha")
	makeRepo(t, root, "beta")
	makeRepo(t, root, "nested/gamma")
	// Worktree — .git is a file, should be skipped.
	makeWorktree(t, root, "alpha-wt")
	// node_modules — should not be recursed.
	os.MkdirAll(filepath.Join(root, "node_modules", "some-pkg", ".git"), 0o755)
	// Too deep (4 levels) — should not be discovered.
	makeRepo(t, root, "a/b/c/d/deep")

	repos, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	names := make(map[string]bool)
	for _, r := range repos {
		names[r.Name] = true
	}

	if !names["alpha"] {
		t.Error("expected alpha to be discovered")
	}
	if !names["beta"] {
		t.Error("expected beta to be discovered")
	}
	if !names["gamma"] {
		t.Error("expected gamma to be discovered")
	}
	if names["alpha-wt"] {
		t.Error("worktree alpha-wt should not be discovered")
	}
	if names["some-pkg"] {
		t.Error("node_modules repo should not be discovered")
	}
	if names["deep"] {
		t.Error("repo 4 levels deep should not be discovered")
	}

	// Verify sorted alphabetically.
	for i := 1; i < len(repos); i++ {
		if repos[i].Name < repos[i-1].Name {
			t.Errorf("repos not sorted: %q before %q", repos[i-1].Name, repos[i].Name)
		}
	}
}

func TestDiscoverMissingRoot(t *testing.T) {
	_, err := Discover("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for missing root, got nil")
	}
}

func TestDisambiguate(t *testing.T) {
	cases := []struct {
		name  string
		input []Repo
		want  map[string]bool // expected names after disambiguation
	}{
		{
			name:  "no collisions",
			input: []Repo{{Name: "alpha", Path: "/a/alpha"}, {Name: "beta", Path: "/b/beta"}},
			want:  map[string]bool{"alpha": true, "beta": true},
		},
		{
			name: "collision",
			input: []Repo{
				{Name: "api", Path: "/work/api"},
				{Name: "api", Path: "/personal/api"},
			},
			want: map[string]bool{"work/api": true, "personal/api": true},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Disambiguate(c.input)
			for _, r := range got {
				if !c.want[r.Name] {
					t.Errorf("unexpected name %q in output", r.Name)
				}
			}
		})
	}
}
