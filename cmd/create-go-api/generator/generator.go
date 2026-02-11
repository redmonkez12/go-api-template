package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/redmonkez12/go-api-template/templates"
)

// Generate creates a new project from the templates using the given config.
func Generate(cfg *ProjectConfig) error {
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	outDir := cfg.ProjectName
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// 1. Copy static files
	if err := copyStatic(outDir, cfg); err != nil {
		return fmt.Errorf("copy static files: %w", err)
	}

	// 2. Copy database variant files
	if err := copyDatabaseVariant(outDir, cfg); err != nil {
		return fmt.Errorf("copy database variant: %w", err)
	}

	// 3. Copy auth variant files
	if err := copyAuthVariant(outDir, cfg); err != nil {
		return fmt.Errorf("copy auth variant: %w", err)
	}

	// 4. Render shared templates
	if err := renderTemplates(outDir, cfg); err != nil {
		return fmt.Errorf("render templates: %w", err)
	}

	return nil
}

// stripGoTmplExt strips the .tmpl suffix from .go.tmpl filenames, returning the .go name.
func stripGoTmplExt(name string) string {
	if strings.HasSuffix(name, ".go.tmpl") {
		return strings.TrimSuffix(name, ".tmpl")
	}
	return name
}

// copyStatic copies all files from templates/static/ to the output directory,
// rewriting Go import paths.
func copyStatic(outDir string, cfg *ProjectConfig) error {
	root := "static"
	return fs.WalkDir(templates.StaticFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path (strip "static/" prefix)
		rel, _ := filepath.Rel(root, path)
		rel = stripGoTmplExt(rel)
		target := filepath.Join(outDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := fs.ReadFile(templates.StaticFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		content := string(data)
		if strings.HasSuffix(path, ".go.tmpl") {
			content = rewriteImports(content, cfg.ModuleName)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(content), 0o644)
	})
}

// copyDatabaseVariant copies the correct database variant files into the output project.
func copyDatabaseVariant(outDir string, cfg *ProjectConfig) error {
	variantRoot := fmt.Sprintf("variants/database/%s/%s", cfg.ORM, cfg.Database)

	return fs.WalkDir(templates.VariantsFS, variantRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(variantRoot, path)
		rel = stripGoTmplExt(rel)
		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(templates.VariantsFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		content := string(data)
		if strings.HasSuffix(path, ".go.tmpl") {
			content = rewriteImports(content, cfg.ModuleName)
		}

		// Determine target path based on filename conventions
		target := resolveVariantTarget(outDir, rel, cfg)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(content), 0o644)
	})
}

// copyAuthVariant copies the correct auth token variant files.
func copyAuthVariant(outDir string, cfg *ProjectConfig) error {
	variantRoot := fmt.Sprintf("variants/auth/%s", cfg.Auth)

	return fs.WalkDir(templates.VariantsFS, variantRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(variantRoot, path)
		rel = stripGoTmplExt(rel)
		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(templates.VariantsFS, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		content := string(data)
		if strings.HasSuffix(path, ".go.tmpl") {
			content = rewriteImports(content, cfg.ModuleName)
		}

		target := filepath.Join(outDir, "internal", "auth", rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(content), 0o644)
	})
}

// resolveVariantTarget maps a variant file to its output path in the generated project.
func resolveVariantTarget(outDir, rel string, cfg *ProjectConfig) string {
	// Migration files go to migrations/
	if strings.HasPrefix(rel, "migrations/") {
		return filepath.Join(outDir, rel)
	}

	// user_repository.go -> internal/user/repository.go
	if rel == "user_repository.go" {
		return filepath.Join(outDir, "internal", "user", "repository.go")
	}

	// auth_repository.go -> internal/auth/repository.go
	if rel == "auth_repository.go" {
		return filepath.Join(outDir, "internal", "auth", "repository.go")
	}

	// models.go -> internal/database/models.go
	if rel == "models.go" {
		return filepath.Join(outDir, "internal", "database", "models.go")
	}

	// DB init files (bun.go, gorm.go, db.go) -> internal/database/
	if rel == "bun.go" || rel == "gorm.go" || rel == "db.go" {
		return filepath.Join(outDir, "internal", "database", rel)
	}

	// Fallback: place in internal/database/
	return filepath.Join(outDir, "internal", "database", rel)
}

// renderTemplates processes .tmpl files from templates/shared/ and writes them to the output.
func renderTemplates(outDir string, cfg *ProjectConfig) error {
	root := "shared"

	tplData := buildTemplateData(cfg)

	return fs.WalkDir(templates.SharedFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		// Strip .tmpl extension for output path
		outPath := strings.TrimSuffix(rel, ".tmpl")
		target := filepath.Join(outDir, outPath)

		data, err := fs.ReadFile(templates.SharedFS, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		tmpl, err := template.New(rel).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		f, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("create %s: %w", target, err)
		}
		defer f.Close()

		return tmpl.Execute(f, tplData)
	})
}

// TemplateData is the data passed to shared .tmpl files.
type TemplateData struct {
	ProjectName string
	ModuleName  string
	Database    Database
	ORM         ORM
	Auth        AuthToken

	// Convenience booleans for templates
	IsPostgres bool
	IsMySQL    bool
	IsMongoDB  bool
	IsBun      bool
	IsGORM     bool
	IsPgx      bool
	IsSQLRaw   bool
	IsMongo    bool
	IsPaseto   bool
	IsJWT      bool
	IsSQL      bool // true for Postgres and MySQL (not MongoDB)
}

func buildTemplateData(cfg *ProjectConfig) *TemplateData {
	return &TemplateData{
		ProjectName: cfg.ProjectName,
		ModuleName:  cfg.ModuleName,
		Database:    cfg.Database,
		ORM:         cfg.ORM,
		Auth:        cfg.Auth,
		IsPostgres:  cfg.Database == DatabasePostgres,
		IsMySQL:     cfg.Database == DatabaseMySQL,
		IsMongoDB:   cfg.Database == DatabaseMongoDB,
		IsBun:       cfg.ORM == ORMBun,
		IsGORM:      cfg.ORM == ORMGORM,
		IsPgx:       cfg.ORM == ORMPgx,
		IsSQLRaw:    cfg.ORM == ORMSQLRaw,
		IsMongo:     cfg.ORM == ORMMongo,
		IsPaseto:    cfg.Auth == AuthPaseto,
		IsJWT:       cfg.Auth == AuthJWT,
		IsSQL:       cfg.Database != DatabaseMongoDB,
	}
}
