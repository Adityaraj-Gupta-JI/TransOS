package cli

import "fmt"

func PrintBanner(version string) {
	fmt.Printf(
		Cyan+"TransOS %s%s\n",
		version,
		Reset,
	)

	fmt.Println(
		Gray +
			"Cross-Platform Environment State & Configuration Migration Engine" +
			Reset,
	)

	fmt.Println()
}

func PrintInteractiveHelp() {
	fmt.Println()

	printSectionHeader("TRANSOS COMMAND REFERENCE", Cyan)

	fmt.Println()
	fmt.Println(Cyan + Bold + "Migration" + Reset)

	fmt.Println(
		"  1 / extract       Capture Windows source state",
	)

	fmt.Println(
		"  2 / validate      Validate migration profile",
	)

	fmt.Println(
		"  3 / preview       Inspect canonical migration JSON",
	)

	fmt.Println(
		"  4 / inject        Generate Linux migration package",
	)

	fmt.Println(
		"  5 / run-all       Extract → Validate → Inject",
	)

	fmt.Println(
		"  6 / rollback      Restore WAL-backed artifact changes",
	)

	fmt.Println()
	fmt.Println(Cyan + Bold + "Inspection" + Reset)

	fmt.Println(
		"  7 / outputs       Show generated migration package",
	)

	fmt.Println(
		"  9 / status        Show migration state and pipeline",
	)

	fmt.Println(
		"  about / 10       Show product and architecture overview",
	)

	fmt.Println(
		"  pwd               Show workspace paths",
	)

	fmt.Println(
		"  dir / ls          Show current directory contents",
	)

	fmt.Println()
	fmt.Println(Cyan + Bold + "Utilities" + Reset)

	fmt.Println(
		"  8 / help          Show this command reference",
	)

	fmt.Println(
		"  menu / home       Redraw the main dashboard",
	)

	fmt.Println(
		"  clear / cls       Clear terminal screen",
	)

	fmt.Println(
		"  version           Show TransOS version",
	)

	fmt.Println(
		"  translate         Show translation-stage information",
	)

	fmt.Println()
	fmt.Println(Cyan + Bold + "Exit" + Reset)

	fmt.Println(
		"  0 / exit / quit / q     Close the TransOS session",
	)

	fmt.Println()
}

func PrintHelp() {
	fmt.Println()

	fmt.Println(
		Cyan +
			Bold +
			"TransOS — Cross-Platform Environment State & Configuration Migration Engine" +
			Reset,
	)

	fmt.Println()

	fmt.Println("Usage:")

	fmt.Println(
		"  transos                       Launch persistent interactive console",
	)

	fmt.Println(
		"  transos extract               Capture Windows source state",
	)

	fmt.Println(
		"  transos validate              Validate migration profile",
	)

	fmt.Println(
		"  transos preview               Preview migration profile",
	)

	fmt.Println(
		"  transos inject [profile]      Generate Linux migration package",
	)

	fmt.Println(
		"  transos import [profile]      Alias for inject",
	)

	fmt.Println(
		"  transos run-all               Extract → Validate → Inject",
	)

	fmt.Println(
		"  transos rollback              Roll back WAL-backed artifact changes",
	)

	fmt.Println(
		"  transos status                Show migration state",
	)

	fmt.Println(
		"  transos outputs               Show generated package",
	)

	fmt.Println(
		"  transos version               Show version information",
	)

	fmt.Println(
		"  transos help                  Show this help",
	)

	fmt.Println()

	fmt.Println(
		Gray +
			"Interactive mode remains active until exit / quit / q / 0." +
			Reset,
	)

	fmt.Println()
}
