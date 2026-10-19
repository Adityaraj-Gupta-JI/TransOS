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
		"  6 / rollback      Roll back the latest or a selected checkpoint",
	)

	fmt.Println()
	fmt.Println(Cyan + Bold + "Configuration" + Reset)
	fmt.Println("  config            View or change mode, extraction depth, and categories")

	fmt.Println()
	fmt.Println(Cyan + Bold + "Migration planning" + Reset)
	fmt.Println("  plan create       Create a review plan; software starts pending")
	fmt.Println("  plan show         List include/exclude decisions and package mappings")
	fmt.Println("  plan include      Approve a software candidate")
	fmt.Println("  plan exclude      Exclude a software candidate")
	fmt.Println("  plan map          Map software to an apt/dnf package name")
	fmt.Println("  plan add/remove   Manage manually added candidates")

	fmt.Println()
	fmt.Println("  package seal      Create SHA-256 manifest for generated artifacts")
	fmt.Println("  package verify    Verify artifacts against the manifest")
	fmt.Println("  package inspect   Show sealed artifact names and hashes")
	fmt.Println("  package target-check  Verify bundle and assess Linux target (read-only)")
	fmt.Println("  package execution-plan  Preview proposed target actions (read-only)")
	fmt.Println("  package stage --approve <destination> [dir]  Copy verified files for manual Linux review")

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
		"  transos extract               Capture source state using saved settings",
	)

	fmt.Println(
		"  transos config                View or change persistent configuration",
	)

	fmt.Println(
		"  transos plan <action>         Create/review decisions for software migration",
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
		"  transos inject --dry-run      Preview readiness without writing files",
	)
	fmt.Println(
		"  transos preflight [profile]   Check profile, plan, secrets and checkpoints",
	)

	fmt.Println("  transos package seal [dir]    Seal generated Linux artifacts")
	fmt.Println("  transos package verify [dir]  Verify sealed artifact integrity")
	fmt.Println("  transos package inspect [dir] Inspect manifest contents")
	fmt.Println("  transos package target-check [dir]  Check Linux target compatibility")
	fmt.Println("  transos package execution-plan [dir]  Preview target actions without execution")
	fmt.Println("  transos package stage --approve <destination> [dir]  Stage verified files on Linux")
	fmt.Println("  transos package apply --approve <bundle> <home> <recovery>  Apply user settings on Linux")
	fmt.Println("  transos package verify-applied <recovery>  Verify applied user settings")
	fmt.Println("  transos package restore --approve <recovery>  Restore original settings")

	fmt.Println(
		"  transos import [profile]      Alias for inject",
	)

	fmt.Println(
		"  transos run-all               Extract → Validate → Inject",
	)

	fmt.Println("  transos rollback              Roll back the latest active checkpoint")
	fmt.Println("  transos rollback list         List retained checkpoints")
	fmt.Println("  transos rollback <checkpoint> Roll back to before that checkpoint")

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
