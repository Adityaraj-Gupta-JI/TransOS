package cli

import (
	"fmt"
	"strings"
)

type helpTopic struct{ name, summary, usage, details, next string }

var helpTopics = []helpTopic{
	{"extract", "Capture Windows configuration and installed software", "transos extract", "Runs extraction using saved configuration. Supported source: Windows.", "transos validate"},
	{"config", "Inspect and change extraction preferences", "transos config [action]", "Controls extraction mode, depth, and categories. Run without arguments for available actions.", "transos extract"},
	{"validate", "Validate the extracted migration profile", "transos validate", "Checks the profile before planning and generating output.", "transos plan create"},
	{"preview", "Inspect the extracted migration profile", "transos preview", "Read the normalized source data before making migration decisions.", "transos plan create"},
	{"plan", "Review and approve software migration decisions", "transos plan <create|show|include|exclude|map|add|remove> [args]", "New software candidates begin pending. Review candidates and explicitly include only what should migrate. Run 'plan' for action usage.", "transos preflight"},
	{"preflight", "Read-only readiness and safety checks", "transos preflight [profile]", "Reports profile, plan, sensitive values, checkpoints, and blockers without generating artifacts.", "transos inject --dry-run"},
	{"inject", "Generate Linux migration artifacts", "transos inject [profile] | transos inject --dry-run [profile]", "Dry-run checks readiness without writes. Normal mode generates shell hooks, environment configuration, and a proposed installer.", "transos package seal"},
	{"run-all", "Run the source-side extraction pipeline", "transos run-all", "Extract, validate, and generate output. Review software decisions separately before approving installation.", "transos package seal"},
	{"package", "Seal, inspect, verify and deploy the migration bundle", "transos package <action> [args]", "Source: seal, inspect, verify. Linux target: target-check, execution-plan, stage, apply, verify-applied, restore. Run 'help package <action>' for exact usage.", "transos package verify"},
	{"rollback", "Recover generated output from retained checkpoints", "transos rollback [list|checkpoint]", "Source-side WAL rollback is different from Linux-side 'package restore'.", "transos status"},
	{"status", "Show migration state and progress", "transos status", "Inspect the active source-side state and determine the next migration step.", "transos preflight"},
	{"outputs", "Inspect generated migration output", "transos outputs", "Lists generated artifacts and their paths.", "transos package seal"},
	{"version", "Display TransOS version", "transos version", "Reports the CLI build version.", "transos help"},
}

var packageHelp = map[string]string{
	"seal":           "transos package seal [dir]  — create the SHA-256 manifest after generating artifacts",
	"verify":         "transos package verify [dir]  — verify sealed bundle checksums (read-only)",
	"inspect":        "transos package inspect [dir]  — show manifest entries (does not verify contents)",
	"target-check":   "transos package target-check [dir]  — inspect Linux target compatibility (read-only)",
	"execution-plan": "transos package execution-plan [dir]  — preview target actions (read-only)",
	"stage":          "transos package stage --approve <destination> [dir]  — copy verified artifacts to a new private Linux directory",
	"apply":          "transos package apply --approve <bundle> <home> <recovery>  — apply user configuration on Linux with recovery records",
	"verify-applied": "transos package verify-applied <recovery>  — check the applied configuration",
	"restore":        "transos package restore --approve <recovery>  — restore prior user configuration on Linux",
}

// printHelpText keeps long command signatures readable in narrow terminals.
func printHelpText(prefix, value string) {
	width := uiWidth() - 6
	for i, line := range uiWrap(value, width) {
		if i == 0 && prefix != "" {
			fmt.Println("  " + prefix)
		}
		fmt.Println("  " + line)
	}
}

func printTopicHelp(args []string) int {
	if len(args) == 0 {
		printCategorizedHelp()
		return 0
	}
	topic := strings.ToLower(strings.TrimSpace(args[0]))
	if topic == "package" && len(args) > 1 {
		action := strings.ToLower(args[1])
		usage, ok := packageHelp[action]
		if !ok {
			fmt.Printf("Unknown package action %q. Try: transos help package\n", args[1])
			return 2
		}
		uiHeader("PACKAGE / "+strings.ToUpper(action), "Command syntax and safety notes")
		uiSection("Usage")
		printHelpText("", strings.Split(usage, "  — ")[0])
		if parts := strings.SplitN(usage, "  — ", 2); len(parts) == 2 {
			uiSection("Behavior")
			printHelpText("", parts[1])
		}
		if action == "apply" || action == "stage" || action == "restore" {
			uiNote("Requires explicit --approve. Review paths and the execution plan first. No software packages are installed automatically.")
		}
		uiFooter("See all actions: transos help package")
		return 0
	}
	if topic == "package" {
		uiHeader("PACKAGE COMMANDS", "Source bundle verification and Linux deployment")
		for _, name := range []string{"seal", "verify", "inspect", "target-check", "execution-plan", "stage", "apply", "verify-applied", "restore"} {
			parts := strings.SplitN(packageHelp[name], "  — ", 2)
			uiSection(name)
			printHelpText("", parts[0])
			if len(parts) > 1 {
				printHelpText("", parts[1])
			}
		}
		uiFooter("Details: transos help package <action>")
		return 0
	}
	for _, t := range helpTopics {
		if t.name == topic {
			uiHeader(strings.ToUpper(t.name), t.summary)
			uiSection("Usage")
			printHelpText("", t.usage)
			uiSection("Description")
			printHelpText("", t.details)
			uiFooter("Next: " + t.next)
			return 0
		}
	}
	if topic == "guide" || topic == "next" {
		uiHeader("GUIDED WORKFLOW", "Read-only migration guidance")
		uiSection("Usage")
		printHelpText("", "transos guide  |  transos next")
		uiNote("Shows current evidence and suggested next steps. Does not run migration operations.")
		return 0
	}
	fmt.Printf("Unknown help topic %q. Use 'transos help' to list categories.\n", topic)
	return 2
}

func printCategorizedHelp() {
	uiHeader("COMMAND GUIDE", "Interactive: transos | Direct: transos <command> [args] | Help: transos help <command>")
	groups := []struct {
		title string
		names []string
	}{
		{"01 SOURCE & SETTINGS", []string{"config", "extract", "validate", "preview"}},
		{"02 MIGRATION WORKFLOW", []string{"plan", "preflight", "inject", "run-all"}},
		{"03 LINUX DEPLOYMENT", []string{"package"}},
		{"04 RECOVERY & INSPECTION", []string{"rollback", "status", "outputs"}},
		{"05 UTILITIES", []string{"version"}},
	}
	for _, g := range groups {
		uiSection(g.title)
		for _, name := range g.names {
			for _, t := range helpTopics {
				if t.name == name {
					fmt.Println("  " + uiPaint("1;36", t.name))
					for _, line := range uiWrap(t.summary, uiWidth()-8) {
						fmt.Println("      " + line)
					}
					break
				}
			}
		}
	}
	uiSection("Read-only guidance")
	printHelpText("", "transos guide | transos next")
	uiNote("Deployment requires Linux and explicit approval. Generated files are not automatically installed.")
	uiFooter("Explore: transos help package | transos help plan")
}
