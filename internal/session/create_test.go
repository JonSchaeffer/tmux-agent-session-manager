package session

import (
	"testing"
)

func TestParseWorktreePath(t *testing.T) {
	porcelain := `worktree /home/user/code/myrepo
HEAD abc123
branch refs/heads/main

worktree /home/user/code/myrepo.feature-auth
HEAD def456
branch refs/heads/feature/auth

worktree /home/user/code/myrepo.fix-bug
HEAD ghi789
branch refs/heads/fix-bug
`
	cases := []struct {
		branch  string
		want    string
		wantErr bool
	}{
		{"main", "/home/user/code/myrepo", false},
		{"feature/auth", "/home/user/code/myrepo.feature-auth", false},
		{"fix-bug", "/home/user/code/myrepo.fix-bug", false},
		{"nonexistent", "", true},
	}
	for _, c := range cases {
		t.Run(c.branch, func(t *testing.T) {
			got, err := parseWorktreePath(porcelain, c.branch)
			if (err != nil) != c.wantErr {
				t.Fatalf("parseWorktreePath() error = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("parseWorktreePath() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestValidBranchRegex(t *testing.T) {
	valid := []string{
		"main",
		"feature/auth",
		"fix-bug_1",
		"release/v1.0",
	}
	invalid := []string{
		"bad branch",
		"feat~1",
		"name:colon",
	}
	for _, b := range valid {
		if !validBranch.MatchString(b) {
			t.Errorf("expected %q to be valid", b)
		}
	}
	for _, b := range invalid {
		if validBranch.MatchString(b) {
			t.Errorf("expected %q to be invalid", b)
		}
	}
}
