package cli

import (
	"fmt"
	"strings"

	"github.com/transos/transos/internal/app"
	"github.com/transos/transos/internal/config"
)

func runConfig(args []string) int {
	cfg, err := app.LoadConfig()
	if err != nil {
		fmt.Printf(Red+"[-] Configuration error: %v\n"+Reset, err)
		fmt.Println(Gray + "    Fix transos_config.json or remove it to recreate defaults." + Reset)
		return 1
	}

	if len(args) == 0 || strings.EqualFold(args[0], "show") {
		printConfig(cfg)
		return 0
	}

	if strings.EqualFold(args[0], "reset") {
		if err := config.Save(config.DefaultPath, config.Default()); err != nil {
			fmt.Printf(Red+"[-] Could not reset configuration: %v\n"+Reset, err)
			return 1
		}
		fmt.Println(Green + "[+] Configuration reset to guided / medium defaults." + Reset)
		return 0
	}

	if !strings.EqualFold(args[0], "set") || len(args) < 3 {
		printConfigHelp()
		return 2
	}

	field := strings.ToLower(strings.TrimSpace(args[1]))
	value := strings.TrimSpace(strings.Join(args[2:], " "))
	switch field {
	case "mode", "operation-mode":
		cfg.OperationMode = strings.ToLower(value)
	case "depth", "extraction-depth":
		cfg.ExtractionDepth = strings.ToLower(value)
	case "categories", "category":
		parts := strings.Split(value, ",")
		cfg.Categories = make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				cfg.Categories = append(cfg.Categories, trimmed)
			}
		}
	default:
		fmt.Printf(Yellow+"[-] Unknown configuration field %q.\n"+Reset, field)
		printConfigHelp()
		return 2
	}

	normalized, err := config.Normalize(cfg)
	if err != nil {
		fmt.Printf(Red+"[-] Invalid configuration: %v\n"+Reset, err)
		return 2
	}
	if err := config.Save(config.DefaultPath, normalized); err != nil {
		fmt.Printf(Red+"[-] Could not save configuration: %v\n"+Reset, err)
		return 1
	}

	fmt.Println(Green + "[+] Configuration saved." + Reset)
	printConfig(normalized)
	return 0
}

func printConfig(cfg config.Config) {
	fmt.Println()
	printSectionHeader("TRANSOS CONFIGURATION", Cyan)
	fmt.Printf("  %-22s %s\n", "Config file:", config.DefaultPath)
	fmt.Printf("  %-22s %s\n", "Operation mode:", cfg.OperationMode)
	fmt.Printf("  %-22s %s\n", "Extraction depth:", cfg.ExtractionDepth)
	fmt.Printf("  %-22s %s\n", "Categories:", strings.Join(cfg.Categories, ", "))
	fmt.Println()
	fmt.Println(Gray + "  Operation mode and extraction depth are independent settings." + Reset)
	fmt.Println(Gray + "  Example: transos config set depth deep" + Reset)
}

func printConfigHelp() {
	fmt.Println()
	fmt.Println("Configuration commands:")
	fmt.Println("  transos config")
	fmt.Println("  transos config show")
	fmt.Println("  transos config set mode automatic|guided|manual")
	fmt.Println("  transos config set depth surface|medium|deep")
	fmt.Println("  transos config set categories environment,software,registry,theme")
	fmt.Println("  transos config reset")
}
