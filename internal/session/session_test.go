package session

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseSessionLine(t *testing.T) {
	now := time.Now().Unix()
	cases := []struct {
		name       string
		line       string
		wantOK     bool
		wantRepo   string
		wantBranch string
	}{
		{
			name:       "simple session",
			line:       fmt.Sprintf("agent/myrepo/main\t%d\t/work/myrepo", now),
			wantOK:     true,
			wantRepo:   "myrepo",
			wantBranch: "main",
		},
		{
			name:       "branch with slash",
			line:       fmt.Sprintf("agent/myrepo/feature/auth\t%d\t/work/myrepo", now),
			wantOK:     true,
			wantRepo:   "myrepo",
			wantBranch: "feature/auth",
		},
		{
			name:   "non-agent session",
			line:   fmt.Sprintf("myproject\t%d\t/work", now),
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
		{
			name:   "too few fields",
			line:   "agent/repo",
			wantOK: false,
		},
		{
			name:   "missing branch segment",
			line:   fmt.Sprintf("agent/onlyone\t%d\t/work", now),
			wantOK: false,
		},
		{
			name:   "invalid timestamp",
			line:   "agent/repo/branch\tnotanumber\t/work",
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, ok := parseSessionLine(c.line)
			if ok != c.wantOK {
				t.Fatalf("parseSessionLine() ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if s.RepoName != c.wantRepo {
				t.Errorf("RepoName = %q, want %q", s.RepoName, c.wantRepo)
			}
			if s.Branch != c.wantBranch {
				t.Errorf("Branch = %q, want %q", s.Branch, c.wantBranch)
			}
		})
	}
}

func TestSanitizeRepoSegment(t *testing.T) {
	cases := []struct{ in, want string }{
		{"work/api", "work-api"},
		{"myrepo", "myrepo"},
		{"a/b/c", "a-b-c"},
	}
	for _, c := range cases {
		got := sanitizeRepoSegment(c.in)
		if got != c.want {
			t.Errorf("sanitizeRepoSegment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeRoundTrip(t *testing.T) {
	disambiguated := "work/api"
	branch := "feature/auth"

	repoSeg := sanitizeRepoSegment(disambiguated)
	sessionName := fmt.Sprintf("agent/%s/%s", repoSeg, branch)

	parts := strings.SplitN(sessionName, "/", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %v", len(parts), parts)
	}
	if parts[1] != "work-api" {
		t.Errorf("repo segment = %q, want work-api", parts[1])
	}
	if parts[2] != "feature/auth" {
		t.Errorf("branch = %q, want feature/auth", parts[2])
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	cases := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "30s ago"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-48 * time.Hour), "2d ago"},
	}
	for _, c := range cases {
		got := RelativeTime(c.t)
		if got != c.want {
			t.Errorf("RelativeTime(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}
