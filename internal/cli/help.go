package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// PrintBanner prints the standard TransOS CLI banner.
func PrintBanner(version string) {
	fmt.Printf(`
████████╗██████╗   █████╗ ███╗   ██╗███████╗ ██████╗ ███████╗
╚══██╔══╝██╔══██╗██╔══██╗████╗  ██║██╔════╝██╔══██╗██╔════╝
   ██║   ██████╔╝███████║██╔██╗ ██║███████╗██║   ██║███████╗
   ██║   ██╔══██╗██╔══██║██║╚██╗██║╚════██║██║   ██║╚════██║
   ██║   ██║  ██║██║  ██║██║ ╚████║███████║╚██████╔╝███████║
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚══════╝
:%s`, version)

	fmt.Println(" OS Configuration Extractor & Translator Tool")
	fmt.Println(" Type 'help' to see all commands or 'exit' to quit.")
	fmt.Println()
}

// PrintStatusSummary prints the current working directory and output path.
func PrintStatusSummary() {
	cwd, _ := os.Getwd()
	outputDir := filepath.Join(cwd, "target_output")

	fmt.Println("------------- [ Environment Info ] -------------")
	fmt.Printf(" Working Directory : %s\n", cwd)
	fmt.Printf(" Saved Outputs Dir : %s\n", outputDir)
	fmt.Println("------------------------------------------------")
}

// PrintOutputInfo describes the currently generated migration artifacts.
func PrintOutputInfo() {
	cwd, _ := os.Getwd()
	outputDir := filepath.Join(cwd, "target_output")

	fmt.Printf("\nTarget Output Folder: %s\n", outputDir)
	fmt.Println("Generated Migration Artifacts:")
	fmt.Println(" ├── install_dependencies.sh  -> Target-side dependency script")
	fmt.Println(" ├── transos_env.conf         -> Generated environment configuration")
	fmt.Println(" ├── .bashrc / .zshrc         -> Generated shell hook artifacts")
	fmt.Println(" └── transos.wal              -> Transaction audit log")
	fmt.Println()
}

// PrintCurrentDirectoryDetails prints the active working directory contents.
func PrintCurrentDirectoryDetails() {
	cwd, _ := os.Getwd()

	fmt.Printf("Current Working Directory: %s\n", cwd)

	files, err := os.ReadDir(cwd)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}

	fmt.Println("Directory Contents:")

	for _, file := range files {
		kind := "<FILE>"

		if file.IsDir() {
			kind = "<DIR> "
		}

		fmt.Printf(" %s  %s\n", kind, file.Name())
	}
}

// PrintInteractiveHelp prints the commands supported by the current CLI.
func PrintInteractiveHelp() {
	fmt.Println(`
TransOS Commands:
  extract        Extract current host configuration
  validate       Validate migration_profile.json
  preview        Preview migration_profile.json
  inject         Generate target migration artifacts
  import         Alias for inject
  rollback       Roll back recorded migration changes
  version        Display TransOS version
  help           Display this help

Compatibility / roadmap commands:
  translate      Not yet a standalone stage
  run-all        Run the current Extract -> Inject pipeline

Information:
  pwd, dir        Display active working directory and contents
  outputs, files  Show generated output information

Exit:
  exit, quit, q   Exit the interactive tool shell`)
}

// PrintHelp prints non-interactive CLI usage.
func PrintHelp() {
	fmt.Println(`Usage:
  transos                         Launch interactive mode
  transos extract                 Extract host state
  transos validate                Validate migration profile
  transos preview                 Preview migration profile
  transos inject [profile]        Generate target migration artifacts
  transos import [profile]        Alias for inject
  transos rollback                Roll back recorded migration changes
  transos version                 Display version information
  transos help                    Show this help

Roadmap / compatibility:
  transos translate               Standalone translation stage (not implemented)
  transos run-all                 Current Extract -> Inject pipeline`)
}
