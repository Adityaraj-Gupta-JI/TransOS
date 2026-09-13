package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/transos/transos/internal/app"
)

const (
	version = "1.0 MVP"

	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Cyan   = "\033[38;2;0;217;255m"
	Blue   = "\033[38;2;22;131;255m"
	Purple = "\033[38;2;181;108;255m"
	Green  = "\033[38;2;0;229;160m"
	Yellow = "\033[38;2;255;216;77m"
	White  = "\033[38;2;245;245;245m"
	Gray   = "\033[38;2;154;164;173m"
)

const banner = `
████████╗██████╗   █████╗ ███╗   ██╗███████╗ ██████╗ ███████╗
╚══██╔══╝██╔══██╗ ██╔══██╗████╗  ██║██╔════╝██╔══██╗██╔════╝
   ██║   ██████╔╝███████║██╔██╗ ██║███████╗██║   ██║███████╗
   ██║   ██╔══██╗██╔══██║██║╚██╗██║╚════██║██║   ██║╚════██║
   ██║   ██║  ██║██║  ██║██║ ╚████║███████║╚██████╔╝███████║
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚══════╝
`

// Run is the single entry point for the TransOS CLI.
func Run(args []string) int {
	if len(args) == 0 {
		return runInteractiveShell()
	}

	command := strings.ToLower(strings.TrimSpace(args[0]))

	switch command {
	case "extract":
		return runExtract()

	case "inject", "import":
		profilePath := app.DefaultProfilePath
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			profilePath = args[1]
		}
		return runInject(profilePath)

	case "rollback":
		return runRollback()

	case "validate":
		return runValidate()

	case "preview":
		return runPreview()

	case "version", "--version", "-v":
		fmt.Println(Cyan + "TransOS Version " + version + Reset)
		return 0

	case "help", "--help", "-h":
		PrintHelp()
		return 0

	case "interactive", "ui":
		return runInteractiveShell()

	case "translate":
		fmt.Println(Yellow + "[-] Standalone translation is not implemented yet." + Reset)
		fmt.Println(Gray + "    Translation currently occurs within the injection pipeline." + Reset)
		return 2

	case "run-all", "all", "--all":
		return runAll()

	case "pwd":
		PrintStatusSummary()
		return 0

	case "dir", "ls":
		PrintCurrentDirectoryDetails()
		return 0

	case "outputs", "files":
		PrintOutputInfo()
		return 0

	case "clear", "cls":
		clearScreen()
		return 0

	default:
		fmt.Printf(Yellow+"[-] Unknown command: '%s'\n"+Reset, command)
		PrintHelp()
		return 1
	}
}

// runInteractiveShell starts the persistent TransOS command shell.
//
// The process remains alive until the user explicitly enters exit, quit, or q,
// or until stdin reaches EOF.
func runInteractiveShell() int {
	clearScreen()
	renderUI()

	fmt.Println()
	fmt.Println(Cyan + Bold + "Interactive Migration Console" + Reset)
	fmt.Println(Gray + "Type 'help' for commands. Type 'exit' to close TransOS." + Reset)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(Green + "transos" + Cyan + "> " + Reset)

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				fmt.Println(Gray + "Input stream closed. Exiting TransOS." + Reset)
				return 0
			}

			fmt.Printf(
				Yellow+"[-] Failed to read command: %v\n"+Reset,
				err,
			)
			return 1
		}

		commandLine := strings.TrimSpace(input)

		if commandLine == "" {
			continue
		}

		if shouldExit(commandLine) {
			fmt.Println()
			fmt.Println(Cyan + "TransOS session closed." + Reset)
			fmt.Println(Gray + "Thank you for using TransOS — Bridging Worlds, Preserving You." + Reset)
			return 0
		}

		runInteractiveCommand(commandLine)
		fmt.Println()
	}
}

func runInteractiveCommand(commandLine string) {
	args := strings.Fields(commandLine)

	if len(args) == 0 {
		return
	}

	// Numeric shortcuts make the dashboard easier to demonstrate live.
	switch strings.ToLower(args[0]) {
	case "1":
		args[0] = "extract"

	case "2":
		args[0] = "validate"

	case "3":
		args[0] = "preview"

	case "4":
		args[0] = "inject"

	case "5":
		args[0] = "run-all"

	case "6":
		args[0] = "rollback"

	case "7":
		args[0] = "outputs"

	case "8":
		args[0] = "help"

	case "9":
		args[0] = "status"

	default:
		// Keep the original command unchanged.
	}

	command := strings.ToLower(args[0])

	switch command {
	case "extract":
		runExtract()

	case "validate":
		runValidate()

	case "preview":
		runPreview()

	case "inject", "import":
		profilePath := app.DefaultProfilePath
		if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
			profilePath = args[1]
		}
		runInject(profilePath)

	case "rollback":
		runRollback()

	case "run-all", "all":
		runAll()

	case "help":
		PrintInteractiveHelp()

	case "version":
		fmt.Println(Cyan + "TransOS Version " + version + Reset)

	case "pwd":
		PrintStatusSummary()

	case "dir", "ls":
		PrintCurrentDirectoryDetails()

	case "outputs", "files":
		PrintOutputInfo()

	case "status":
		printInteractiveStatus()

	case "menu":
		clearScreen()
		renderUI()

	case "clear", "cls":
		clearScreen()
		renderUI()

	case "translate":
		fmt.Println(
			Yellow + "[-] Standalone translation is not implemented yet." +
				Reset,
		)
		fmt.Println(
			Gray +
				"    Translation currently occurs within the injection pipeline." +
				Reset,
		)

	default:
		fmt.Printf(
			Yellow+"[-] Unknown command: '%s'\n"+Reset,
			command,
		)
		fmt.Println(
			Gray +
				"    Type 'help' to see available commands." +
				Reset,
		)
	}
}

func shouldExit(command string) bool {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "exit", "quit", "q", "0":
		return true
	default:
		return false
	}
}

func printInteractiveStatus() {
	fmt.Println(Cyan + Bold + "Current TransOS Session" + Reset)
	fmt.Println(Gray + "----------------------------------------" + Reset)

	if _, err := os.Stat(app.DefaultProfilePath); err == nil {
		fmt.Println(Green + "  Migration Profile : READY" + Reset)
	} else {
		fmt.Println(Yellow + "  Migration Profile : NOT GENERATED" + Reset)
	}

	if _, err := os.Stat(app.DefaultOutputDir); err == nil {
		fmt.Println(Green + "  Target Output     : AVAILABLE" + Reset)
	} else {
		fmt.Println(Yellow + "  Target Output     : NOT GENERATED" + Reset)
	}

	if _, err := os.Stat(app.DefaultWALPath); err == nil {
		fmt.Println(Green + "  WAL               : AVAILABLE" + Reset)
	} else {
		fmt.Println(Gray + "  WAL               : NOT YET CREATED" + Reset)
	}

	fmt.Printf(
		Gray+"  Source runtime    : %s/%s\n"+Reset,
		runtime.GOOS,
		runtime.GOARCH,
	)

	fmt.Println()
}

func runExtract() int {
	fmt.Println(Cyan + "[*] Mode: Real Extraction Engine Active..." + Reset)

	if err := app.ExtractProfile(app.DefaultProfilePath); err != nil {
		fmt.Printf(
			Yellow+"[-] Extraction failed: %v\n"+Reset,
			err,
		)
		return 1
	}

	profile, err := app.LoadProfile(app.DefaultProfilePath)
	if err != nil {
		fmt.Printf(
			Yellow+"[-] Extraction completed, but the generated profile could not be reloaded: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Println()
	fmt.Println(Cyan + Bold + "Extraction Summary" + Reset)
	fmt.Printf(
		Green+"  Environment Variables : %d\n"+Reset,
		len(profile.Environment),
	)
	fmt.Printf(
		Green+"  Software Entries      : %d\n"+Reset,
		len(profile.Software),
	)
	fmt.Printf(
		Green+"  Registry Entries      : %d\n"+Reset,
		len(profile.Registry),
	)

	fmt.Printf(
		Green+"  Schema                : %s\n"+Reset,
		profile.Metadata.SchemaVersion,
	)

	fmt.Printf(
		Green+"  Profile               : %s\n"+Reset,
		app.DefaultProfilePath,
	)

	fmt.Println()
	fmt.Println(
		Gray +
			"Source state captured successfully. Use 'validate', 'preview', or 'inject' next." +
			Reset,
	)

	return 0
}

func runInject(profilePath string) int {
	fmt.Println(
		Cyan +
			"[*] Mode: Real Injection & Persistence Engine Active..." +
			Reset,
	)

	if err := app.InjectProfile(profilePath); err != nil {
		fmt.Printf(
			Yellow+"[-] Injection failed: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Println(
		Green +
			"[+] Injection pipeline completed. Migration artifacts generated." +
			Reset,
	)

	PrintOutputInfo()

	return 0
}

func runRollback() int {
	fmt.Println(Cyan + "[*] Rolling back TransOS transaction..." + Reset)

	if err := app.Rollback(app.DefaultWALPath); err != nil {
		fmt.Printf(
			Yellow+"[-] Rollback failed using %s: %v\n"+Reset,
			app.DefaultWALPath,
			err,
		)
		return 1
	}

	fmt.Println(
		Green + "[+] Rollback completed successfully." + Reset,
	)

	return 0
}

func runValidate() int {
	fmt.Println(Cyan + "[*] Validating migration profile..." + Reset)

	if err := app.ValidateProfile(app.DefaultProfilePath); err != nil {
		fmt.Printf(
			Yellow+"[-] Profile invalid: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Printf(
		Green+"[+] Profile %s is valid!\n"+Reset,
		app.DefaultProfilePath,
	)

	return 0
}

func runPreview() int {
	fmt.Println(Cyan + "[*] Previewing migration profile..." + Reset)

	data, err := app.PreviewProfile(app.DefaultProfilePath)
	if err != nil {
		fmt.Printf(
			Yellow+"[-] Cannot preview profile: %v\n"+Reset,
			err,
		)
		return 1
	}

	var formatted json.RawMessage

	if err := json.Unmarshal(data, &formatted); err != nil {
		fmt.Printf(
			Yellow+"[-] Profile contains invalid JSON: %v\n"+Reset,
			err,
		)
		return 1
	}

	pretty, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		fmt.Printf(
			Yellow+"[-] Failed to format profile: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Println(White + string(pretty) + Reset)

	return 0
}

func runAll() int {
	fmt.Println(Cyan + "[*] Running current TransOS migration pipeline..." + Reset)
	fmt.Println(Gray + "    Extract -> Validate -> Inject" + Reset)
	fmt.Println()

	if code := runExtract(); code != 0 {
		return code
	}

	fmt.Println()

	if code := runValidate(); code != 0 {
		return code
	}

	fmt.Println()

	if code := runInject(app.DefaultProfilePath); code != 0 {
		return code
	}

	fmt.Println(
		Green + "[+] Current migration pipeline completed." + Reset,
	)

	return 0
}

func renderUI() {
	fmt.Println()
	fmt.Println(Blue + Bold + banner + Reset)

	fmt.Println(
		Cyan +
			Bold +
			centerText(
				"B R I D G I N G   W O R L D S,   P R E S E R V I N G   Y O U",
				118,
			) +
			Reset,
	)

	fmt.Println()

	fmt.Println(
		Gray +
			centerText(
				"Automated Cross-Platform Environment State and Configuration Migrator",
				118,
			) +
			Reset,
	)

	fmt.Println()

	aboutLines := []string{
		"TransOS is an automated, lightweight, cross-platform",
		"environment migration engine designed to extract",
		"source configuration into a portable migration profile.",
		"",
		"It separates source extraction from target-side",
		"migration so Windows state can be analyzed before use.",
		"",
		Cyan + Bold + `"Same You. Different OS. No Friction."` + Reset,
	}

	systemLines := []string{
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"OS",
			Reset,
			White+runtime.GOOS+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Architecture",
			Reset,
			White+runtime.GOARCH+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Runtime",
			Reset,
			White+"Go"+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Profile",
			Reset,
			White+app.DefaultProfilePath+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Output",
			Reset,
			White+app.DefaultOutputDir+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Status",
			Reset,
			Green+Bold+"● READY"+Reset,
		),
	}

	featureLines := []string{
		Green + Bold + "✓ " + Reset + "Extract Windows environment and system state",
		Green + Bold + "✓ " + Reset + "Capture installed software and registry state",
		Green + Bold + "✓ " + Reset + "Generate canonical migration profile",
		Green + Bold + "✓ " + Reset + "Translate Windows paths for Linux",
		Green + Bold + "✓ " + Reset + "Generate Linux migration artifacts",
		Green + Bold + "✓ " + Reset + "Transactional WAL-backed file generation",
		Green + Bold + "✓ " + Reset + "Persistent interactive migration console",
		"",
		Gray + "○ Full software compatibility analyzer   [NEXT]" + Reset,
		Gray + "○ Full migration planner                 [NEXT]" + Reset,
		Gray + "○ Verification and recovery engine       [NEXT]" + Reset,
	}

	projectLines := []string{
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Version",
			Reset,
			White+version+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Language",
			Reset,
			White+"Go"+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Profile",
			Reset,
			White+app.DefaultProfilePath+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Output",
			Reset,
			White+app.DefaultOutputDir+Reset,
		),
		fmt.Sprintf(
			"%s%-15s%s : %s",
			Cyan+Bold,
			"Target",
			Reset,
			White+"Linux migration package"+Reset,
		),
	}

	commandLines := []string{
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"1  extract",
			Reset,
			"Capture source state",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"2  validate",
			Reset,
			"Validate migration profile",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"3  preview",
			Reset,
			"Inspect captured state",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"4  inject",
			Reset,
			"Generate Linux artifacts",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"5  run-all",
			Reset,
			"Run Extract -> Validate -> Inject",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"6  rollback",
			Reset,
			"Restore WAL-backed changes",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"7  outputs",
			Reset,
			"Show generated package",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"8  help",
			Reset,
			"Show command reference",
		),
		fmt.Sprintf(
			"%s%-22s%s : %s",
			Green+Bold,
			"0  exit",
			Reset,
			"Close interactive shell",
		),
	}

	quickStartLines := []string{
		Cyan + Bold + "Recommended demo flow" + Reset,
		"",
		Green + "1" + Reset + "  run-all",
		Gray + "   Extract -> Validate -> Generate Linux package" + Reset,
		"",
		Green + "2" + Reset + "  outputs",
		Gray + "   Inspect generated migration artifacts" + Reset,
		"",
		Green + "3" + Reset + "  exit",
		Gray + "   Continue on the Linux VM" + Reset,
	}

	aboutBox := createBox(
		"ⓘ  About TransOS",
		Blue,
		aboutLines,
		65,
	)

	systemBox := createBox(
		"▣  System Information",
		Cyan,
		systemLines,
		50,
	)

	featuresBox := createBox(
		"⚙  Key Features  [V1.0 MVP]",
		Purple,
		featureLines,
		65,
	)

	projectBox := createBox(
		"◉  Project Details",
		Green,
		projectLines,
		50,
	)

	commandsBox := createBox(
		"❯  Interactive Commands",
		Yellow,
		commandLines,
		50,
	)

	quickStartBox := createBox(
		"⚡  Quick Start",
		Cyan,
		quickStartLines,
		50,
	)

	topRow := joinHorizontal(aboutBox, systemBox)

	rightColumn := append(projectBox, commandsBox...)
	rightColumn = append(rightColumn, quickStartBox...)

	bottomRow := joinHorizontal(featuresBox, rightColumn)

	for _, line := range topRow {
		fmt.Println(line)
	}

	fmt.Println()

	for _, line := range bottomRow {
		fmt.Println(line)
	}

	fmt.Println()

	fmt.Println(
		Cyan +
			Bold +
			"──── TRANSOS — BRIDGING WORLDS, PRESERVING YOU ────" +
			Reset,
	)

	fmt.Println()
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func createBox(
	title string,
	colorCode string,
	lines []string,
	width int,
) []string {
	var box []string

	topWidth := width - len(title) - 4
	if topWidth < 1 {
		topWidth = 1
	}

	topBorder := colorCode +
		"╭─ " +
		title +
		" " +
		strings.Repeat("─", topWidth) +
		"╮" +
		Reset

	box = append(box, topBorder)

	for _, line := range lines {
		visibleLen := stripANSIWidth(line)
		padding := width - visibleLen - 4

		if padding < 0 {
			padding = 0
		}

		content := colorCode + "│ " + Reset +
			line +
			strings.Repeat(" ", padding) +
			colorCode + " │" + Reset

		box = append(box, content)
	}

	bottomBorder := colorCode +
		"╰" +
		strings.Repeat("─", width-2) +
		"╯" +
		Reset

	box = append(box, bottomBorder)

	return box
}

func stripANSIWidth(s string) int {
	inEscape := false
	length := 0

	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}

		if inEscape {
			if (r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		length++
	}

	return length
}

func joinHorizontal(left, right []string) []string {
	maxLen := len(left)

	if len(right) > maxLen {
		maxLen = len(right)
	}

	var combined []string

	leftWidth := 0
	if len(left) > 0 {
		leftWidth = stripANSIWidth(left[0])
	}

	for i := 0; i < maxLen; i++ {
		lLine := ""
		rLine := ""

		if i < len(left) {
			lLine = left[i]
		} else {
			lLine = strings.Repeat(" ", leftWidth)
		}

		if i < len(right) {
			rLine = right[i]
		}

		combined = append(combined, lLine+"  "+rLine)
	}

	return combined
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}

	pad := (width - len(text)) / 2

	return strings.Repeat(" ", pad) + text
}
