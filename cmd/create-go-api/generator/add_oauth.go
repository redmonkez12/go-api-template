package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// AddOAuth adds OAuth support to an existing generated project using a
// generate-diff-apply strategy: it generates two temporary projects
// (with and without OAuth), diffs them, and applies the patch.
func AddOAuth(projectDir string) error {
	// 1. Load existing config
	cfg, err := LoadConfigFromFile(projectDir)
	if err != nil {
		return fmt.Errorf("not a create-go-api project (missing %s): %w", ConfigFileName, err)
	}

	// 2. Validate OAuth not already enabled
	if cfg.HasOAuth {
		return fmt.Errorf("OAuth is already enabled in this project")
	}

	// 3. Verify git repo
	if _, err := os.Stat(filepath.Join(projectDir, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("git repository required — run 'git init' first")
	}

	// 4. Create temp dir with a/ and b/ subdirs
	tmpDir, err := os.MkdirTemp("", "go-api-oauth-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")

	// 5. Generate without OAuth (baseline)
	cfgWithout := *cfg
	cfgWithout.HasOAuth = false
	if err := GenerateTo(dirA, &cfgWithout); err != nil {
		return fmt.Errorf("generate baseline project: %w", err)
	}

	// 6. Generate with OAuth
	cfgWith := *cfg
	cfgWith.HasOAuth = true
	if err := GenerateTo(dirB, &cfgWith); err != nil {
		return fmt.Errorf("generate oauth project: %w", err)
	}

	// 7. Run diff
	patch, err := generateDiff(tmpDir)
	if err != nil {
		return fmt.Errorf("generate diff: %w", err)
	}

	if len(bytes.TrimSpace(patch)) == 0 {
		return fmt.Errorf("no differences found — OAuth may already be integrated")
	}

	// 8. Renumber migration files if needed
	patch, err = fixMigrationNumbers(patch, projectDir)
	if err != nil {
		return fmt.Errorf("fix migration numbers: %w", err)
	}

	// 9. Apply patch
	if err := applyPatch(projectDir, patch); err != nil {
		// Save patch for manual review
		patchPath := filepath.Join(projectDir, "oauth.patch")
		_ = os.WriteFile(patchPath, patch, 0o644)
		return fmt.Errorf("patch failed to apply cleanly (saved to oauth.patch for manual review): %w", err)
	}

	// 10. Update config
	cfg.HasOAuth = true
	if err := cfg.SaveToFile(projectDir); err != nil {
		return fmt.Errorf("update config: %w", err)
	}

	// 11. Run go mod tidy
	if err := runGoModTidy(projectDir); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}

// generateDiff runs diff -ruN between a/ and b/ in the given directory.
func generateDiff(tmpDir string) ([]byte, error) {
	cmd := exec.Command("diff", "-ruN", "--exclude=go.sum", "a", "b")
	cmd.Dir = tmpDir

	out, err := cmd.CombinedOutput()
	// diff exits 1 when files differ (which is expected)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return out, nil
		}
		return nil, fmt.Errorf("diff command failed: %w\n%s", err, out)
	}
	// exit 0 means no differences
	return out, nil
}

// applyPatch applies a unified diff patch using git apply.
func applyPatch(projectDir string, patch []byte) error {
	cmd := exec.Command("git", "apply", "-p1", "--stat", "--apply")
	cmd.Dir = projectDir
	cmd.Stdin = bytes.NewReader(patch)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, out)
	}
	return nil
}

// fixMigrationNumbers adjusts OAuth migration numbers in the patch if 000003 is
// already taken by a user migration in the project.
func fixMigrationNumbers(patch []byte, projectDir string) ([]byte, error) {
	nextNum, err := nextMigrationNumber(projectDir)
	if err != nil {
		// No migrations dir — use default numbering
		return patch, nil
	}

	// OAuth migrations in the generated diff use 000003
	if nextNum <= 3 {
		// 000003 is available, no renumbering needed
		return patch, nil
	}

	oldPrefix := "000003"
	newPrefix := fmt.Sprintf("%06d", nextNum)
	patch = bytes.ReplaceAll(patch, []byte(oldPrefix+"_add_oauth_fields"), []byte(newPrefix+"_add_oauth_fields"))

	return patch, nil
}

// nextMigrationNumber finds the highest existing migration number + 1.
func nextMigrationNumber(projectDir string) (int, error) {
	migrationsDir := filepath.Join(projectDir, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, err
	}

	maxNum := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Migration files are named like 000001_create_users_table.up.sql
		idx := strings.IndexByte(name, '_')
		if idx < 1 {
			continue
		}
		num, err := strconv.Atoi(name[:idx])
		if err != nil {
			continue
		}
		if num > maxNum {
			maxNum = num
		}
	}

	return maxNum + 1, nil
}

// runGoModTidy runs go mod tidy in the project directory.
func runGoModTidy(projectDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, out)
	}
	return nil
}

// MigrationFiles returns sorted migration filenames from the project's migrations directory.
// Exported for testing.
func MigrationFiles(projectDir string) ([]string, error) {
	migrationsDir := filepath.Join(projectDir, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}
