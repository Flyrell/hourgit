package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldIgnoreGitDir(t *testing.T) {
	assert.True(t, ShouldIgnore("/repo", "/repo/.git/objects/pack"))
	assert.True(t, ShouldIgnore("/repo", "/repo/.git/HEAD"))
	assert.True(t, ShouldIgnore("/repo", "/repo/.git"))
}

func TestShouldIgnoreNormalFiles(t *testing.T) {
	assert.False(t, ShouldIgnore("/repo", "/repo/main.go"))
	assert.False(t, ShouldIgnore("/repo", "/repo/src/app.go"))
}

func TestShouldIgnoreGitignorePatterns(t *testing.T) {
	repo := t.TempDir()
	gitignore := "node_modules\n*.log\nbuild/\n"
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(gitignore), 0644))

	assert.True(t, ShouldIgnore(repo, filepath.Join(repo, "node_modules", "pkg", "index.js")))
	assert.True(t, ShouldIgnore(repo, filepath.Join(repo, "app.log")))
	assert.True(t, ShouldIgnore(repo, filepath.Join(repo, "build", "output.js")))
	assert.False(t, ShouldIgnore(repo, filepath.Join(repo, "src", "main.go")))
}

func TestShouldIgnoreWithPatterns(t *testing.T) {
	repo := t.TempDir()
	patterns := []string{"node_modules", "*.log", "build"}

	assert.True(t, ShouldIgnoreWithPatterns(repo, filepath.Join(repo, "node_modules", "pkg", "index.js"), patterns))
	assert.True(t, ShouldIgnoreWithPatterns(repo, filepath.Join(repo, "app.log"), patterns))
	assert.True(t, ShouldIgnoreWithPatterns(repo, filepath.Join(repo, "build", "output.js"), patterns))
	assert.False(t, ShouldIgnoreWithPatterns(repo, filepath.Join(repo, "src", "main.go"), patterns))
	// .git is always excluded regardless of patterns
	assert.True(t, ShouldIgnoreWithPatterns(repo, filepath.Join(repo, ".git", "HEAD"), nil))
}

func TestLoadGitignorePatterns(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("node_modules\n# comment\n\n*.log\n"), 0644))

	patterns := LoadGitignorePatterns(repo)
	assert.Equal(t, []string{"node_modules", "*.log"}, patterns)
}

func TestLoadGitignorePatternsNoFile(t *testing.T) {
	repo := t.TempDir()
	patterns := LoadGitignorePatterns(repo)
	assert.Nil(t, patterns)
}

func TestMatchPatternWildcard(t *testing.T) {
	assert.True(t, matchPattern("foo.log", "*.log"))
	assert.False(t, matchPattern("foo.txt", "*.log"))
}

func TestMatchPatternExactName(t *testing.T) {
	assert.True(t, matchPattern("node_modules/pkg/file.js", "node_modules"))
	assert.False(t, matchPattern("src/main.go", "node_modules"))
}

func TestMatchPatternDirSlash(t *testing.T) {
	assert.True(t, matchPattern("build/output.js", "build"))
}

func TestMatchPatternWithSlash(t *testing.T) {
	assert.True(t, matchPattern("dist/bundle.js", "dist/*"))
}

func TestMatchPatternNegation(t *testing.T) {
	// Negation patterns are skipped (not supported)
	assert.False(t, matchPattern("important.log", "!important.log"))
}
