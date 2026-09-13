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
	Red    = "\033[38;2;255;92;92m"
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

type sessionState struct {
	profileReady bool
	outputReady  bool
	walReady     bool

	environmentCount int
	softwareCount    int
	registryCount    int

	profileValid bool
}

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
		printTranslationNotice()
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

	case "status", "state":
		printInteractiveStatus()
		return 0

	case "about":
		renderAbout()
		return 0

	case "clear", "cls":
		clearScreen()
		return 0

	default:
		fmt.Printf(
			Yellow+"[-] Unknown command: '%s'\n"+Reset,
			command,
		)

		fmt.Println(
			Gray + "    Type 'help' to see available commands." + Reset,
		)

		return 1
	}
}

func runInteractiveShell() int {
	clearScreen()
	renderUI()

	fmt.Println(
		Cyan + Bold + "Interactive Migration Console" + Reset,
	)

	fmt.Println(
		Gray +
			"Commands can be entered by name or number. Type 'help' for help." +
			Reset,
	)

	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(interactivePrompt())

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				fmt.Println(
					Gray + "Input stream closed. Exiting TransOS." + Reset,
				)
				return 0
			}

			fmt.Printf(
				Red+"[-] Failed to read command: %v\n"+Reset,
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
			fmt.Println(
				Cyan + "TransOS session closed." + Reset,
			)
			fmt.Println(
				Gray +
					"Thank you for using TransOS — Bridging Worlds, Preserving You." +
					Reset,
			)

			return 0
		}

		runInteractiveCommand(commandLine)

		fmt.Println()
	}
}

func interactivePrompt() string {
	state := readSessionState()

	profile := Gray + "P○" + Reset
	output := Gray + "O○" + Reset
	wal := Gray + "W○" + Reset

	if state.profileReady {
		profile = Green + "P✓" + Reset
	}

	if state.outputReady {
		output = Green + "O✓" + Reset
	}

	if state.walReady {
		wal = Green + "W✓" + Reset
	}

	return Green +
		"transos" +
		Reset +
		" [" +
		profile +
		" " +
		output +
		" " +
		wal +
		"]" +
		Cyan +
		"> " +
		Reset
}

func runInteractiveCommand(commandLine string) {
	args := strings.Fields(commandLine)

	if len(args) == 0 {
		return
	}

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
	case "10":
		args[0] = "about"
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
		fmt.Println(
			Cyan + "TransOS Version " + version + Reset,
		)

	case "pwd":
		PrintStatusSummary()

	case "dir", "ls":
		PrintCurrentDirectoryDetails()

	case "outputs", "files":
		PrintOutputInfo()

	case "status", "state":
		printInteractiveStatus()

	case "about":
		renderAbout()

	case "menu", "home":
		clearScreen()
		renderUI()

	case "clear", "cls":
		clearScreen()

	case "translate":
		printTranslationNotice()

	default:
		fmt.Printf(
			Yellow+"[-] Unknown command: '%s'\n"+Reset,
			command,
		)

		fmt.Println(
			Gray +
				"    Type 'help' for commands or 'menu' to redraw the dashboard." +
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

func readSessionState() sessionState {
	state := sessionState{}

	if _, err := os.Stat(app.DefaultProfilePath); err == nil {
		state.profileReady = true
	}

	if _, err := os.Stat(app.DefaultOutputDir); err == nil {
		state.outputReady = true
	}

	if _, err := os.Stat(app.DefaultWALPath); err == nil {
		state.walReady = true
	}

	profile, err := app.LoadProfile(app.DefaultProfilePath)
	if err == nil {
		state.profileValid = true
		state.environmentCount = len(profile.Environment)
		state.softwareCount = len(profile.Software)
		state.registryCount = len(profile.Registry)
	}

	return state
}

func printInteractiveStatus() {
	state := readSessionState()

	fmt.Println()
	printSectionHeader("CURRENT MIGRATION STATE", Cyan)

	printStateRow(
		"Source",
		runtime.GOOS+"/"+runtime.GOARCH,
		Cyan,
	)

	if state.profileValid {
		printStateRow(
			"Profile",
			fmt.Sprintf(
				"VALID • %d env • %d software • %d registry",
				state.environmentCount,
				state.softwareCount,
				state.registryCount,
			),
			Green,
		)
	} else if state.profileReady {
		printStateRow(
			"Profile",
			"PRESENT • validation required",
			Yellow,
		)
	} else {
		printStateRow(
			"Profile",
			"NOT GENERATED",
			Gray,
		)
	}

	if state.outputReady {
		printStateRow(
			"Package",
			"GENERATED • target_output/",
			Green,
		)
	} else {
		printStateRow(
			"Package",
			"NOT GENERATED",
			Gray,
		)
	}

	if state.walReady {
		printStateRow(
			"WAL",
			"AVAILABLE • transactional history present",
			Green,
		)
	} else {
		printStateRow(
			"WAL",
			"NOT CREATED",
			Gray,
		)
	}

	fmt.Println()

	printPipeline(state)

	fmt.Println()
}

func printPipeline(state sessionState) {
	printSectionHeader("MIGRATION PIPELINE", Purple)

	extractStatus := Gray + "○" + Reset
	validateStatus := Gray + "○" + Reset
	packageStatus := Gray + "○" + Reset

	if state.profileReady {
		extractStatus = Green + "✓" + Reset
	}

	if state.profileValid {
		validateStatus = Green + "✓" + Reset
	}

	if state.outputReady {
		packageStatus = Green + "✓" + Reset
	}

	fmt.Printf(
		"  %s EXTRACT  %s  %s VALIDATE  %s  %s PACKAGE\n",
		extractStatus,
		Cyan+"→"+Reset,
		validateStatus,
		Cyan+"→"+Reset,
		packageStatus,
	)

	fmt.Printf(
		"  %s ANALYZE  %s  %s PLAN  %s  %s APPLY  %s  %s VERIFY\n",
		Gray+"○"+Reset,
		Cyan+"·"+Reset,
		Gray+"○"+Reset,
		Cyan+"·"+Reset,
		Gray+"○"+Reset,
		Cyan+"·"+Reset,
		Gray+"○"+Reset,
	)
}

func printStateRow(label, value, colorCode string) {
	fmt.Printf(
		"  %s%-12s%s : %s%s%s\n",
		Cyan+Bold,
		label,
		Reset,
		colorCode,
		value,
		Reset,
	)
}

func renderUI() {
	state := readSessionState()

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
				"Cross-Platform Environment State & Configuration Migration Engine",
				118,
			) +
			Reset,
	)

	fmt.Println()

	sessionLines := buildSessionLines(state)
	pipelineLines := buildPipelineLines(state)
	commandLines := buildCommandLines()
	quickStartLines := buildQuickStartLines(state)
	aboutLines := buildAboutLines()
	projectLines := buildProjectLines()

	sessionBox := createBox(
		"◉  Session",
		Cyan,
		sessionLines,
		57,
	)

	pipelineBox := createBox(
		"⇄  Migration Pipeline",
		Purple,
		pipelineLines,
		58,
	)

	commandsBox := createBox(
		"❯  Commands",
		Yellow,
		commandLines,
		57,
	)

	quickStartBox := createBox(
		"⚡  Quick Start",
		Blue,
		quickStartLines,
		58,
	)

	aboutBox := createBox(
		"ⓘ  About TransOS",
		Blue,
		aboutLines,
		57,
	)

	projectBox := createBox(
		"▣  Project",
		Green,
		projectLines,
		58,
	)

	topRow := joinHorizontal(sessionBox, pipelineBox)
	middleRow := joinHorizontal(commandsBox, quickStartBox)
	bottomRow := joinHorizontal(aboutBox, projectBox)

	for _, line := range topRow {
		fmt.Println(line)
	}

	fmt.Println()

	for _, line := range middleRow {
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
			"──── TRANSOS • WINDOWS → LINUX • PRESERVE STATE, NOT THE OS ────" +
			Reset,
	)

	fmt.Println()
}

func buildSessionLines(state sessionState) []string {
	profileStatus := Gray + "NOT READY" + Reset
	outputStatus := Gray + "NOT READY" + Reset
	walStatus := Gray + "NOT READY" + Reset

	if state.profileValid {
		profileStatus = Green + "VALID" + Reset
	} else if state.profileReady {
		profileStatus = Yellow + "PRESENT" + Reset
	}

	if state.outputReady {
		outputStatus = Green + "READY" + Reset
	}

	if state.walReady {
		walStatus = Green + "AVAILABLE" + Reset
	}

	return []string{
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Cyan+Bold,
			"Source",
			Reset,
			White+runtime.GOOS+"/"+runtime.GOARCH+Reset,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Cyan+Bold,
			"Profile",
			Reset,
			profileStatus,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Cyan+Bold,
			"Package",
			Reset,
			outputStatus,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Cyan+Bold,
			"WAL",
			Reset,
			walStatus,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Cyan+Bold,
			"Schema",
			Reset,
			White+"2.0.0"+Reset,
		),
	}
}

func buildPipelineLines(state sessionState) []string {
	extract := "○ pending"
	validate := "○ pending"
	packageState := "○ pending"

	if state.profileReady {
		extract = Green + "✓ complete" + Reset
	}

	if state.profileValid {
		validate = Green + "✓ complete" + Reset
	}

	if state.outputReady {
		packageState = Green + "✓ complete" + Reset
	}

	return []string{
		"  01  Extract       " + extract,
		"  02  Normalize    " + stateLine("built into profile", Green),
		"  03  Validate      " + validate,
		"  04  Translate     " + stateLine("during generation", Cyan),
		"  05  Package       " + packageState,
		"  06  Apply         " + stateLine("Linux target", Gray),
		"  07  Verify        " + stateLine("next phase", Gray),
	}
}

func buildCommandLines() []string {
	return []string{
		"  1  extract       Capture Windows state",
		"  2  validate      Validate migration profile",
		"  3  preview       Inspect captured JSON",
		"  4  inject        Generate Linux package",
		"  5  run-all       Extract → Validate → Inject",
		"  6  rollback      Restore WAL-backed changes",
		"  7  outputs       Show generated artifacts",
		"  8  help          Command reference",
		"  9  status        Current migration state",
		" 10  about         Product / architecture",
		"  0  exit          Close TransOS",
	}
}

func buildQuickStartLines(state sessionState) []string {
	if !state.profileReady {
		return []string{
			Cyan + Bold + "First run" + Reset,
			"",
			Green + "1" + Reset + "  extract",
			Gray + "   Capture Windows environment" + Reset,
			"",
			Green + "2" + Reset + "  validate",
			Gray + "   Check canonical profile" + Reset,
			"",
			Green + "3" + Reset + "  inject",
			Gray + "   Generate Linux package" + Reset,
		}
	}

	if !state.outputReady {
		return []string{
			Cyan + Bold + "Continue migration" + Reset,
			"",
			Green + "1" + Reset + "  validate",
			Gray + "   Confirm profile integrity" + Reset,
			"",
			Green + "2" + Reset + "  inject",
			Gray + "   Generate target package" + Reset,
			"",
			Green + "3" + Reset + "  outputs",
			Gray + "   Inspect generated files" + Reset,
		}
	}

	return []string{
		Cyan + Bold + "Package ready" + Reset,
		"",
		Green + "1" + Reset + "  outputs",
		Gray + "   Inspect target_output/" + Reset,
		"",
		Green + "2" + Reset + "  exit",
		Gray + "   Continue on Linux VM" + Reset,
		"",
		Green + "3" + Reset + "  rollback",
		Gray + "   Restore previous state" + Reset,
	}
}

func buildAboutLines() []string {
	return []string{
		"TransOS separates source discovery from",
		"target-side migration through a canonical",
		"migration profile.",
		"",
		"Windows → profile → translation →",
		"Linux migration package.",
		"",
		Cyan + Bold + `"Same You. Different OS. No Friction."` + Reset,
	}
}

func buildProjectLines() []string {
	return []string{
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Green+Bold,
			"Version",
			Reset,
			White+version+Reset,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Green+Bold,
			"Language",
			Reset,
			White+"Go"+Reset,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Green+Bold,
			"Profile",
			Reset,
			White+app.DefaultProfilePath+Reset,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Green+Bold,
			"Output",
			Reset,
			White+app.DefaultOutputDir+Reset,
		),
		fmt.Sprintf(
			"%s%-13s%s : %s",
			Green+Bold,
			"Target",
			Reset,
			White+"Linux"+Reset,
		),
	}
}

func renderAbout() {
	fmt.Println()
	printSectionHeader("ABOUT TRANSOS", Cyan)

	fmt.Println(
		White +
			"TransOS is a cross-platform environment state and" +
			Reset,
	)

	fmt.Println(
		White +
			"configuration migration engine designed around a" +
			Reset,
	)

	fmt.Println(
		White +
			"portable canonical migration profile." +
			Reset,
	)

	fmt.Println()

	fmt.Println(
		Cyan + "Architecture:" + Reset,
	)

	fmt.Println(
		Gray +
			"  Discovery → Extraction → Normalization → Translation" +
			Reset,
	)

	fmt.Println(
		Gray +
			"  → Packaging → Linux Apply → Verification" +
			Reset,
	)

	fmt.Println()

	fmt.Println(
		Gray +
			"Current MVP: Windows extraction, canonical profile," +
			Reset,
	)

	fmt.Println(
		Gray +
			"semantic path translation, Linux package generation," +
			Reset,
	)

	fmt.Println(
		Gray +
			"and WAL-backed artifact operations." +
			Reset,
	)

	fmt.Println()
}

func printSectionHeader(title string, colorCode string) {
	const width = 70

	titleWidth := stripANSIWidth(title)

	repeatWidth := width - titleWidth - 5

	if repeatWidth < 1 {
		repeatWidth = 1
	}

	fmt.Println(
		colorCode +
			Bold +
			"╭─ " +
			title +
			" " +
			strings.Repeat("─", repeatWidth) +
			"╮" +
			Reset,
	)
}

func stateLine(value, colorCode string) string {
	return colorCode + value + Reset
}

func printTranslationNotice() {
	fmt.Println(
		Yellow +
			"[-] Standalone translation is not exposed as a separate command yet." +
			Reset,
	)

	fmt.Println(
		Gray +
			"    Semantic translation currently participates in migration generation." +
			Reset,
	)
}

func runExtract() int {
	fmt.Println()
	printSectionHeader("EXTRACTION", Cyan)

	fmt.Println(
		Gray + "  Capturing Windows environment and configuration state..." + Reset,
	)

	if err := app.ExtractProfile(app.DefaultProfilePath); err != nil {
		fmt.Printf(
			Red+"[-] Extraction failed: %v\n"+Reset,
			err,
		)
		return 1
	}

	profile, err := app.LoadProfile(app.DefaultProfilePath)
	if err != nil {
		fmt.Printf(
			Red+
				"[-] Extraction completed, but the generated profile could not be reloaded: %v\n"+
				Reset,
			err,
		)
		return 1
	}

	fmt.Println()

	printResult(
		"Environment variables",
		fmt.Sprintf("%d discovered", len(profile.Environment)),
		Green,
	)

	printResult(
		"Software entries",
		fmt.Sprintf("%d discovered", len(profile.Software)),
		Green,
	)

	printResult(
		"Registry entries",
		fmt.Sprintf("%d discovered", len(profile.Registry)),
		Green,
	)

	printResult(
		"Canonical schema",
		profile.Metadata.SchemaVersion,
		Green,
	)

	printResult(
		"Migration profile",
		app.DefaultProfilePath,
		Green,
	)

	fmt.Println()
	fmt.Println(
		Cyan +
			"→ Next:" +
			Reset +
			" validate the generated profile.",
	)

	return 0
}

func runInject(profilePath string) int {
	fmt.Println()
	printSectionHeader("PACKAGE GENERATION", Cyan)

	fmt.Println(
		Gray + "  Translating source state into Linux migration artifacts..." + Reset,
	)

	if err := app.InjectProfile(profilePath); err != nil {
		fmt.Printf(
			Red+"[-] Injection failed: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Println()

	printResult(
		"Environment configuration",
		"generated",
		Green,
	)

	printResult(
		"Shell hook artifacts",
		"generated",
		Green,
	)

	printResult(
		"Linux dependency installer",
		"generated",
		Green,
	)

	printResult(
		"WAL transaction",
		"recorded",
		Green,
	)

	fmt.Println()
	fmt.Println(
		Cyan +
			"→ Next:" +
			Reset +
			" transfer target_output/ to the Linux system.",
	)

	return 0
}

func runRollback() int {
	fmt.Println()
	printSectionHeader("ROLLBACK", Yellow)

	if err := app.Rollback(app.DefaultWALPath); err != nil {
		fmt.Printf(
			Red+"[-] Rollback failed using %s: %v\n"+Reset,
			app.DefaultWALPath,
			err,
		)
		return 1
	}

	fmt.Println()
	fmt.Println(
		Green + "✓ Transaction rollback completed." + Reset,
	)

	return 0
}

func runValidate() int {
	fmt.Println()
	printSectionHeader("VALIDATION", Cyan)

	fmt.Println(
		Gray + "  Checking canonical migration profile..." + Reset,
	)

	if err := app.ValidateProfile(app.DefaultProfilePath); err != nil {
		fmt.Printf(
			Red+"[-] Profile invalid: %v\n"+Reset,
			err,
		)
		return 1
	}

	profile, err := app.LoadProfile(app.DefaultProfilePath)
	if err != nil {
		fmt.Printf(
			Red+"[-] Profile could not be loaded: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Println()

	printResult(
		"Profile",
		"VALID",
		Green,
	)

	printResult(
		"Schema",
		profile.Metadata.SchemaVersion,
		Green,
	)

	printResult(
		"Environment",
		fmt.Sprintf("%d entries", len(profile.Environment)),
		Green,
	)

	printResult(
		"Software",
		fmt.Sprintf("%d entries", len(profile.Software)),
		Green,
	)

	printResult(
		"Registry",
		fmt.Sprintf("%d entries", len(profile.Registry)),
		Green,
	)

	return 0
}

func runPreview() int {
	fmt.Println()
	printSectionHeader("PROFILE PREVIEW", Cyan)

	data, err := app.PreviewProfile(app.DefaultProfilePath)
	if err != nil {
		fmt.Printf(
			Red+"[-] Cannot preview profile: %v\n"+Reset,
			err,
		)
		return 1
	}

	var formatted json.RawMessage

	if err := json.Unmarshal(data, &formatted); err != nil {
		fmt.Printf(
			Red+"[-] Profile contains invalid JSON: %v\n"+Reset,
			err,
		)
		return 1
	}

	pretty, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		fmt.Printf(
			Red+"[-] Failed to format profile: %v\n"+Reset,
			err,
		)
		return 1
	}

	fmt.Println(White + string(pretty) + Reset)

	return 0
}

func runAll() int {
	fmt.Println()
	printSectionHeader("FULL MIGRATION PIPELINE", Cyan)

	fmt.Println(
		Gray +
			"  Extract → Validate → Generate Linux Package" +
			Reset,
	)

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

	fmt.Println()
	fmt.Println(
		Green +
			Bold +
			"✓ TransOS migration package generation completed." +
			Reset,
	)

	return 0
}

func printResult(label, value, colorCode string) {
	fmt.Printf(
		"  %s%-26s%s %s%s%s\n",
		Cyan,
		label,
		Reset,
		colorCode,
		value,
		Reset,
	)
}

func PrintStatusSummary() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf(
			Red+"[-] Cannot determine working directory: %v\n"+Reset,
			err,
		)
		return
	}

	fmt.Println()
	printSectionHeader("WORKSPACE", Cyan)

	fmt.Printf(
		"  %sWorking directory%s : %s\n",
		Cyan+Bold,
		Reset,
		cwd,
	)

	fmt.Printf(
		"  %sProfile path%s      : %s\n",
		Cyan+Bold,
		Reset,
		app.DefaultProfilePath,
	)

	fmt.Printf(
		"  %sOutput directory%s  : %s\n",
		Cyan+Bold,
		Reset,
		app.DefaultOutputDir,
	)

	fmt.Println()
}

func PrintOutputInfo() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf(
			Red+"[-] Cannot determine working directory: %v\n"+Reset,
			err,
		)
		return
	}

	outputDir := app.DefaultOutputDir

	fmt.Println()
	printSectionHeader("GENERATED MIGRATION PACKAGE", Green)

	fmt.Printf(
		"  %sLocation%s : %s\n",
		Green+Bold,
		Reset,
		cwd+string(os.PathSeparator)+outputDir,
	)

	fmt.Println()

	artifacts := []string{
		"install_dependencies.sh   Linux dependency/application installer",
		"transos_env.conf          Translated environment configuration",
		".bashrc                   Bash integration artifact",
		".zshrc                    Zsh integration artifact",
		"transos.wal               Transaction audit / rollback log",
	}

	for _, artifact := range artifacts {
		fmt.Println(
			"  " +
				Green +
				"✓" +
				Reset +
				" " +
				artifact,
		)
	}

	fmt.Println()
}

func PrintCurrentDirectoryDetails() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf(
			Red+"[-] Error reading directory: %v\n"+Reset,
			err,
		)
		return
	}

	files, err := os.ReadDir(cwd)
	if err != nil {
		fmt.Printf(
			Red+"[-] Error reading directory: %v\n"+Reset,
			err,
		)
		return
	}

	fmt.Println()
	printSectionHeader("WORKING DIRECTORY", Cyan)

	fmt.Println(
		"  " +
			Gray +
			cwd +
			Reset,
	)

	fmt.Println()

	for _, file := range files {
		marker := "├──"

		if file.IsDir() {
			marker = "└─┬"
		}

		fmt.Printf(
			"  %s %s%s%s\n",
			Gray,
			marker,
			Reset,
			file.Name(),
		)
	}

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
	if width < 20 {
		width = 20
	}

	var box []string

	titleWidth := stripANSIWidth(title)

	// Fixed top-border characters:
	// "╭─ " = 3
	// " "   = 1
	// "╮"   = 1
	// Therefore the repeat section is width - title - 5.
	topWidth := width - titleWidth - 5

	if topWidth < 1 {
		topWidth = 1
	}

	topBorder :=
		colorCode +
			"╭─ " +
			title +
			" " +
			strings.Repeat("─", topWidth) +
			"╮" +
			Reset

	box = append(box, topBorder)

	for _, line := range lines {
		visibleLen := stripANSIWidth(line)

		// Content consists of:
		// "│ " = 2
		// " │" = 2
		padding := width - visibleLen - 4

		if padding < 0 {
			padding = 0
		}

		content :=
			colorCode +
				"│ " +
				Reset +
				line +
				strings.Repeat(" ", padding) +
				colorCode +
				" │" +
				Reset

		box = append(box, content)
	}

	bottomBorder :=
		colorCode +
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

	leftWidth := 0

	if len(left) > 0 {
		leftWidth = stripANSIWidth(left[0])
	}

	var combined []string

	for i := 0; i < maxLen; i++ {
		leftLine := ""
		rightLine := ""

		if i < len(left) {
			leftLine = left[i]
		} else {
			leftLine = strings.Repeat(" ", leftWidth)
		}

		if i < len(right) {
			rightLine = right[i]
		}

		combined = append(
			combined,
			leftLine+"  "+rightLine,
		)
	}

	return combined
}

func centerText(text string, width int) string {
	visibleWidth := stripANSIWidth(text)

	if visibleWidth >= width {
		return text
	}

	padding := (width - visibleWidth) / 2

	return strings.Repeat(" ", padding) + text
}
