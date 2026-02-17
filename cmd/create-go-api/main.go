package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redmonkez12/go-api-template/cmd/create-go-api/generator"
	"github.com/redmonkez12/go-api-template/cmd/create-go-api/ui"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "create-go-api",
		Short: "Generate a production-ready Go REST API project",
		Long:  "Interactive CLI to scaffold a Go REST API with your choice of database, ORM, and auth strategy.",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		RunE:  runCreate,
	}

	// Flags for non-interactive mode (CI/scripting)
	createCmd.Flags().String("name", "", "Project name")
	createCmd.Flags().String("module", "", "Go module name")
	createCmd.Flags().String("database", "", "Database (postgres, mysql, mongodb)")
	createCmd.Flags().String("orm", "", "ORM/driver (bun, gorm, pgx, sqlraw, mongo)")
	createCmd.Flags().String("auth", "", "Auth token strategy (paseto, jwt)")
	createCmd.Flags().Bool("oauth", false, "Include OAuth support (Google, GitHub, Discord)")

	// add command group
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add features to an existing project",
	}

	addOAuthCmd := &cobra.Command{
		Use:   "oauth",
		Short: "Add OAuth support (Google, GitHub, Discord) to an existing project",
		RunE:  runAddOAuth,
	}
	addOAuthCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	addCmd.AddCommand(addOAuthCmd)
	rootCmd.AddCommand(createCmd, addCmd)

	// Allow running without subcommand (default to create)
	rootCmd.RunE = createCmd.RunE
	rootCmd.Flags().AddFlagSet(createCmd.Flags())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runAddOAuth(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if !yes {
		fmt.Println("This will add OAuth support (Google, GitHub, Discord) to your project.")
		fmt.Print("Continue? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println("Adding OAuth support...")
	if err := generator.AddOAuth(cwd); err != nil {
		ui.PrintError(err.Error())
		return err
	}

	ui.PrintAddOAuthSuccess()
	return nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	module, _ := cmd.Flags().GetString("module")
	database, _ := cmd.Flags().GetString("database")
	orm, _ := cmd.Flags().GetString("orm")
	auth, _ := cmd.Flags().GetString("auth")
	oauth, _ := cmd.Flags().GetBool("oauth")

	// If all required flags are provided, run non-interactively
	if name != "" && module != "" && database != "" && orm != "" && auth != "" {
		cfg := &generator.ProjectConfig{
			ProjectName: name,
			ModuleName:  module,
			Database:    generator.Database(database),
			ORM:         generator.ORM(orm),
			Auth:        generator.AuthToken(auth),
			HasOAuth:    oauth,
		}

		fmt.Printf("Generating project %q...\n", cfg.ProjectName)
		if err := generator.Generate(cfg); err != nil {
			ui.PrintError(err.Error())
			return err
		}

		ui.PrintSuccess(cfg)
		return nil
	}

	// Interactive mode
	fmt.Println()
	fmt.Println("  Go API Template Generator")
	fmt.Println()

	cfg, err := ui.RunForm()
	if err != nil {
		return fmt.Errorf("form cancelled: %w", err)
	}

	ui.PrintSummary(cfg)

	fmt.Println("Generating project...")
	if err := generator.Generate(cfg); err != nil {
		ui.PrintError(err.Error())
		return err
	}

	ui.PrintSuccess(cfg)
	return nil
}
