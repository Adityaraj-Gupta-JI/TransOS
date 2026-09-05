package cli

import (
	"encoding/json"
	"fmt"
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
╚══██╔══╝██╔══██╗██╔══██╗████╗  ██║██╔════╝██╔══██╗██╔════╝
   ██║   ██████╔╝███████║██╔██╗ ██║███████╗██║   ██║███████╗
   ██║   ██╔══██╗██╔══██║██║╚██╗██║╚════██║██║   ██║╚════██║
   ██║   ██║  ██║██║  ██║██║ ╚████║███████║╚██████╔╝███████║
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚══════╝
`

// Run is the single entry point for the TransOS command-line interface.
//
// The cmd package performs process startup only. CLI interaction and command
// routing live in this package, while application orchestration lives in the
// internal/app package.
func Run(args []string) int {
	if len(args) == 0 {
		renderUI()
		return 0
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
		renderUI()
		return 0

	case "translate":
		fmt.Println(Yellow + "[-] Standalone translation is not implemented yet." + Reset)
		fmt.Println(Gray + "    Translation currently occurs within the injection pipeline." + Reset)
		return 2

	case "run-all", "all", "--all":
		return runAll()

	default:
		fmt.Printf(Yellow+"[-] Unknown command: '%s'\n"+Reset, command)
		PrintHelp()
		return 1
	}
}

func runExtract() int {
	fmt.Println(Cyan + "[*] Mode: Real Extraction Engine Active..." + Reset)

	if err := app.ExtractProfile(app.DefaultProfilePath); err != nil {
		fmt.Printf(Yellow+"[-] Extraction failed: %v\n"+Reset, err)
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

	fmt.Printf(
		Green+"[+] Success! Harvested %d environment variables. Dumped to: %s\n"+Reset,
		len(profile.Environment),
		app.DefaultProfilePath,
	)

	return 0
}

func runInject(profilePath string) int {
	fmt.Println(Cyan + "[*] Mode: Real Injection & Persistence Engine Active..." + Reset)

	if err := app.InjectProfile(profilePath); err != nil {
		fmt.Printf(Yellow+"[-] Injection failed: %v\n"+Reset, err)
		return 1
	}

	fmt.Println(
		Green +
			"[+] Injection pipeline completed. Migration artifacts generated." +
			Reset,
	)

	return 0
}

func runRollback() int {
	if err := app.Rollback(app.DefaultWALPath); err != nil {
		fmt.Printf(
			Yellow+"[-] Rollback failed using %s: %v\n"+Reset,
			app.DefaultWALPath,
			err,
		)
		return 1
	}

	fmt.Println(Green + "[+] Rollback completed successfully." + Reset)
	return 0
}

func runValidate() int {
	fmt.Println(Cyan + "[*] Validating migration profile..." + Reset)

	if err := app.ValidateProfile(app.DefaultProfilePath); err != nil {
		fmt.Printf(Yellow+"[-] Profile invalid: %v\n"+Reset, err)
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
		fmt.Printf(Yellow+"[-] Cannot preview profile: %v\n"+Reset, err)
		return 1
	}

	var formatted json.RawMessage
	if err := json.Unmarshal(data, &formatted); err != nil {
		fmt.Printf(Yellow+"[-] Profile contains invalid JSON: %v\n"+Reset, err)
		return 1
	}

	pretty, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		fmt.Printf(Yellow+"[-] Failed to format profile: %v\n"+Reset, err)
		return 1
	}

	fmt.Println(White + string(pretty) + Reset)
	return 0
}

func runAll() int {
	fmt.Println(Cyan + "[*] Running current TransOS migration pipeline..." + Reset)
	fmt.Println(Gray + "    Extract -> Inject" + Reset)

	if code := runExtract(); code != 0 {
		return code
	}

	if code := runInject(app.DefaultProfilePath); code != 0 {
		return code
	}

	fmt.Println(Green + "[+] Current migration pipeline completed." + Reset)
	return 0
}

func renderUI() {
	fmt.Println()
	fmt.Println(Blue + Bold + banner + Reset)
	fmt.Println(Cyan + Bold + centerText("B R I D G I N G   W O R L D S,   P R E S E R V I N G   Y O U", 118) + Reset)
	fmt.Println()
	fmt.Println(Gray + centerText("Automated Cross-Platform Environment State and Configuration Migrator", 118) + Reset)
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
	aboutBox := createBox("ⓘ  About TransOS", Blue, aboutLines, 65)

	systemLines := []string{
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "OS", Reset, White+runtime.GOOS+" "+runtime.GOARCH+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Architecture", Reset, White+runtime.GOARCH+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Shell", Reset, White+"PowerShell / Bash"+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Terminal", Reset, White+"Standard Console"+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "TransOS", Reset, White+"Migration Engine"+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Status", Reset, Green+Bold+"● Ready"+Reset),
	}
	systemBox := createBox("▣  System Information", Cyan, systemLines, 50)

	featureLines := []string{
		Green + Bold + "✓ " + Reset + "Extract Windows environment variables and PATH",
		Green + Bold + "✓ " + Reset + "Generate migration_profile.json",
		Green + Bold + "✓ " + Reset + "Validate migration profile",
		Green + Bold + "✓ " + Reset + "Preview captured migration state",
		Green + Bold + "✓ " + Reset + "Generate target migration artifacts",
		Green + Bold + "✓ " + Reset + "Transactional rollback prototype",
		Green + Bold + "✓ " + Reset + "Zero external runtime dependencies",
		"",
		Gray + "○ Application compatibility engine       [NEXT]" + Reset,
		Gray + "○ Browser profiles                       [FUTURE]" + Reset,
		Gray + "○ Hardware / device migration            [FUTURE]" + Reset,
	}
	featuresBox := createBox("⚙  Key Features  [V1.0 MVP]", Purple, featureLines, 65)

	projectLines := []string{
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Version", Reset, White+version+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Language", Reset, White+"Go"+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Architecture", Reset, White+"Application Layer"+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Profile", Reset, White+app.DefaultProfilePath+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Output", Reset, White+app.DefaultOutputDir+Reset),
		fmt.Sprintf("%s%-15s%s : %s", Cyan+Bold, "Target", Reset, White+"Linux migration target"+Reset),
	}
	projectBox := createBox("◉  Project Details", Green, projectLines, 50)

	commandLines := []string{
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos", Reset, "Start interactive mode"),
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos extract", Reset, "Extract source environment"),
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos validate", Reset, "Validate migration profile"),
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos preview", Reset, "Preview migration profile"),
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos inject", Reset, "Generate migration artifacts"),
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos rollback", Reset, "Rollback recorded changes"),
		fmt.Sprintf("%s%-22s%s : %s", Green+Bold, "transos version", Reset, "Show version information"),
	}
	commandsBox := createBox("❯  Available Commands", Yellow, commandLines, 50)

	topRow := joinHorizontal(aboutBox, systemBox)
	rightColumn := append(projectBox, commandsBox...)
	bottomRow := joinHorizontal(featuresBox, rightColumn)

	for _, line := range topRow {
		fmt.Println(line)
	}

	fmt.Println()

	for _, line := range bottomRow {
		fmt.Println(line)
	}

	fmt.Println()
	fmt.Println(Cyan + Bold + "──── TRANSOS — BRIDGING WORLDS, PRESERVING YOU ────" + Reset)
	fmt.Println()
}

func createBox(title, colorCode string, lines []string, width int) []string {
	var box []string

	topWidth := width - len(title) - 4
	if topWidth < 1 {
		topWidth = 1
	}

	topBorder := colorCode + "╭─ " + title + " " + strings.Repeat("─", topWidth) + "╮" + Reset
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

	bottomBorder := colorCode + "╰" + strings.Repeat("─", width-2) + "╯" + Reset
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
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
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
