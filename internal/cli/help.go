package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/transos/transos/internal/app"
)

func PrintBanner(version string) {
	fmt.Printf(`
	
████████╗██████╗   █████╗ ███╗   ██╗███████╗ ██████╗ ███████╗
╚══██╔══╝██╔══██╗██╔══██╗████╗  ██║██╔════╝██╔═══██╗██╔════╝
   ██║   ██████╔╝███████║██╔██╗ ██║███████╗██║   ██║███████╗
   ██║   ██╔══██╗██╔══██║██║╚██╗██║╚════██║██║   ██║╚════██║
   ██║   ██║  ██║██║  ██║██║ ╚████║███████║╚██████╔╝███████║
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚══════╝
:%s`, version)
	fmt.Println(" OS Configuration Extractor & Translator Tool")
	fmt.Println(" Type 'help' to see all commands or 'exit' to quit.")
	fmt.Println()
}

func PrintStatusSummary() {
	cwd, _ := os.Getwd()
	outputDir := filepath.Join(cwd, "target_output")

	fmt.Println("------------- [ Environment Info ] -------------")
	fmt.Printf(" Working Directory : %s\n", cwd)
	fmt.Printf(" Saved Outputs Dir : %s\n", outputDir)
	fmt.Println("------------------------------------------------")
}

func PrintOutputInfo() {
	cwd, _ := os.Getwd()
	outputDir := filepath.Join(cwd, "target_output")

	fmt.Printf("\nTarget Output Folder: %s\n", outputDir)
	fmt.Println("Generated Artifacts & Linux Execution Commands:")
	fmt.Println(" ├── install_dependencies.sh  -> Run on target: bash target_output/install_dependencies.sh")
	fmt.Println(" ├── transos_env.conf        -> Source on target: source target_output/transos_env.conf")
	fmt.Println(" ├── .bashrc / .zshrc        -> Append/merge with user shell configs")
	fmt.Println(" └── transos.wal             -> Transaction audit log file")
	fmt.Println()
}

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

func PrintInteractiveHelp() {
	fmt.Println(`
Interactive Shell Commands:
  1, extract      : Extract current OS configurations (Windows/Shell)
  2, translate    : Translate extracted configs to target Linux format
  3, inject       : Generate Linux shell scripts (.bashrc, .zshrc, install scripts)
  4, run-all      : Execute complete pipeline (Extract -> Translate -> Inject)
  
  pwd, dir        : Display active working directory and contents
  outputs, files  : Show saved output file paths & Linux execution instructions
  help, h, ?      : Display this help context
  exit, quit, q   : Exit the interactive tool shell`)
}

func PrintHelp() {
	fmt.Println(`Usage: transos [OPTIONS] or run without options for Interactive Mode.

Options:
  --extract     Run extraction module only
  --translate   Run translation module only
  --inject      Run target script injection module only
  --all         Execute full pipeline
  --version     Display version info
  --help        Show this help message`)
}

func RunDirectFlags(extract, translate, inject, all bool) {
	if all {
		app.RunExtraction()
		app.RunTranslation()
		app.RunInjection()
		return
	}
	if extract {
		app.RunExtraction()
	}
	if translate {
		app.RunTranslation()
	}
	if inject {
		app.RunInjection()
	}
}
