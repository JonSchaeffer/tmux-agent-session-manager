package repo

import (
	"os"
	"path/filepath"
	"sort"
)

type Repo struct {
	Name string
	Path string
}

func Discover(root string) ([]Repo, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "discover", Path: root, Err: os.ErrInvalid}
	}

	var repos []Repo
	err = walkRepos(root, root, 0, 3, &repos)
	if err != nil {
		return nil, err
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	return repos, nil
}

func walkRepos(root, dir string, depth, maxDepth int, repos *[]Repo) error {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // skip unreadable directories
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == ".git" || name == "node_modules" {
			continue
		}

		fullPath := filepath.Join(dir, name)

		// Check if this directory is a git repo.
		// A real repo has .git as a directory; a worktree has .git as a file
		// (containing a gitdir: pointer). Skip worktrees.
		gitPath := filepath.Join(fullPath, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				*repos = append(*repos, Repo{
					Name: filepath.Base(fullPath),
					Path: fullPath,
				})
			}
			// Don't recurse into git repos or worktrees.
			continue
		}

		walkRepos(root, fullPath, depth+1, maxDepth, repos)
	}
	return nil
}

func Disambiguate(repos []Repo) []Repo {
	// Count occurrences of each name.
	counts := make(map[string]int, len(repos))
	for _, r := range repos {
		counts[r.Name]++
	}

	result := make([]Repo, len(repos))
	for i, r := range repos {
		if counts[r.Name] > 1 {
			parent := filepath.Base(filepath.Dir(r.Path))
			result[i] = Repo{
				Name: parent + "/" + r.Name,
				Path: r.Path,
			}
		} else {
			result[i] = r
		}
	}

	// Re-sort after disambiguation since names changed.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
