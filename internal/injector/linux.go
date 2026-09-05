package injector

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/transos/transos/internal/schema"
	"github.com/transos/transos/internal/translator"
	"github.com/transos/transos/internal/wal"
)

const (
	DefaultOutputDir = "target_output"
	DefaultWALPath   = "target_output/transos.wal"
)

// InjectProfile generates the current TransOS migration artifacts from a
// canonical migration profile.
//
// This stage currently generates a portable migration bundle rather than
// mutating the live Linux system. Direct host application will be introduced
// in the Linux application phase.
func InjectProfile(profile *schema.MigrationProfile) error {
	if profile == nil {
		return fmt.Errorf("migration profile is nil")
	}

	if err := profile.Validate(); err != nil {
		return fmt.Errorf("validate migration profile: %w", err)
	}

	outputDir := DefaultOutputDir

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	tx, err := wal.NewTransaction(outputDir)
	if err != nil {
		return fmt.Errorf("create WAL transaction: %w", err)
	}

	if err := injectEnvironment(tx, profile, outputDir); err != nil {
		return err
	}

	if _, err := InjectShellHooks(tx, outputDir); err != nil {
		return fmt.Errorf("inject shell hooks: %w", err)
	}

	if err := injectSoftwareDependencies(tx, profile, outputDir); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration transaction: %w", err)
	}

	fmt.Printf(
		"Migration artifacts generated successfully: %s\n",
		outputDir,
	)

	return nil
}

func injectEnvironment(
	tx *wal.WALTransaction,
	profile *schema.MigrationProfile,
	outputDir string,
) error {
	path := outputDir + string(os.PathSeparator) + "transos_env.conf"

	var builder strings.Builder

	builder.WriteString("# TransOS generated environment configuration\n")
	builder.WriteString("# Source OS: ")
	builder.WriteString(profile.SourceSystem.OS)
	builder.WriteString("\n")
	builder.WriteString("# Schema: ")
	builder.WriteString(profile.Metadata.SchemaVersion)
	builder.WriteString("\n\n")

	environment := append([]schema.EnvironmentVar(nil), profile.Environment...)

	sort.Slice(environment, func(i, j int) bool {
		return environment[i].Name < environment[j].Name
	})

	for _, env := range environment {
		if !isValidShellIdentifier(env.Name) {
			continue
		}

		value := env.Value

		switch env.Type {
		case schema.EnvironmentTypePath,
			schema.EnvironmentTypePathList,
			schema.EnvironmentTypeDirectory,
			schema.EnvironmentTypeFile:
			value = translator.TranslatePathString(value)
		}

		builder.WriteString("export ")
		builder.WriteString(env.Name)
		builder.WriteString("=")
		builder.WriteString(shellQuote(value))
		builder.WriteString("\n")
	}

	if err := tx.LogAction(
		wal.ActionCreateFile,
		path,
		builder.String(),
	); err != nil {
		return fmt.Errorf("log environment artifact: %w", err)
	}

	return nil
}

func injectSoftwareDependencies(
	tx *wal.WALTransaction,
	profile *schema.MigrationProfile,
	outputDir string,
) error {
	path := outputDir + string(os.PathSeparator) + "install_dependencies.sh"

	script := translator.GenerateDependencyScript(profile.Software)

	if err := tx.LogAction(
		wal.ActionCreateFile,
		path,
		script,
	); err != nil {
		return fmt.Errorf("log dependency script: %w", err)
	}

	return nil
}

func isValidShellIdentifier(name string) bool {
	if name == "" {
		return false
	}

	for index, char := range name {
		if index == 0 {
			if !((char >= 'A' && char <= 'Z') ||
				(char >= 'a' && char <= 'z') ||
				char == '_') {
				return false
			}
			continue
		}

		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	return true
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
