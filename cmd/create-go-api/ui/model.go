package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/redmonkez12/go-api-template/cmd/create-go-api/generator"
)

// RunForm displays the interactive project setup form and returns a ProjectConfig.
func RunForm() (*generator.ProjectConfig, error) {
	var (
		projectName string
		moduleName  string
		database    string
		orm         string
		auth        string
	)

	// Stage 1: Project info + database selection
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Description("Directory name for the new project").
				Placeholder("my-api").
				Value(&projectName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("project name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Go module name").
				Description("e.g. github.com/yourname/my-api").
				Placeholder("github.com/yourname/my-api").
				Value(&moduleName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("module name is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Database").
				Options(
					huh.NewOption("PostgreSQL", string(generator.DatabasePostgres)),
					huh.NewOption("MySQL", string(generator.DatabaseMySQL)),
					huh.NewOption("MongoDB", string(generator.DatabaseMongoDB)),
				).
				Value(&database),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form1.Run(); err != nil {
		return nil, err
	}

	// Stage 2: ORM selection (depends on database choice)
	db := generator.Database(database)
	ormOptions := buildORMOptions(db)

	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("ORM / Driver").
				Options(ormOptions...).
				Value(&orm),

			huh.NewSelect[string]().
				Title("Auth token strategy").
				Options(
					huh.NewOption("PASETO v4 (recommended)", string(generator.AuthPaseto)),
					huh.NewOption("JWT (HS256)", string(generator.AuthJWT)),
				).
				Value(&auth),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form2.Run(); err != nil {
		return nil, err
	}

	cfg := &generator.ProjectConfig{
		ProjectName: strings.TrimSpace(projectName),
		ModuleName:  strings.TrimSpace(moduleName),
		Database:    db,
		ORM:         generator.ORM(orm),
		Auth:        generator.AuthToken(auth),
	}

	return cfg, nil
}

// PrintSummary prints the selected configuration.
func PrintSummary(cfg *generator.ProjectConfig) {
	fmt.Println(titleStyle.Render("Project Configuration"))
	fmt.Printf("  Project:  %s\n", cfg.ProjectName)
	fmt.Printf("  Module:   %s\n", cfg.ModuleName)
	fmt.Printf("  Database: %s\n", cfg.Database.Label())
	fmt.Printf("  ORM:      %s\n", cfg.ORM.Label())
	fmt.Printf("  Auth:     %s\n", cfg.Auth.Label())
	fmt.Println()
}

// PrintSuccess prints the success message with next steps.
func PrintSuccess(cfg *generator.ProjectConfig) {
	fmt.Println(successStyle.Render("Project created successfully!"))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", cfg.ProjectName)
	fmt.Println("  cp .env.example .env")
	fmt.Println("  # Edit .env with your configuration")
	if cfg.Database != generator.DatabaseMongoDB {
		fmt.Println("  make docker-up")
		fmt.Println("  make migrate-up")
	} else {
		fmt.Println("  make docker-up")
	}
	fmt.Println("  make run")
	fmt.Println()
}

// PrintError prints an error message.
func PrintError(msg string) {
	fmt.Println(errorStyle.Render("Error: " + msg))
}

func buildORMOptions(db generator.Database) []huh.Option[string] {
	orms := generator.ORMsForDatabase(db)
	opts := make([]huh.Option[string], 0, len(orms))
	for _, o := range orms {
		opts = append(opts, huh.NewOption(o.Label(), string(o)))
	}
	return opts
}
