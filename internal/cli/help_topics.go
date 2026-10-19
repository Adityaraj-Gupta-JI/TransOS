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

func printTopicHelp(args []string) int {
	if len(args) == 0 {
		printCategorizedHelp()
		return 0
	}
	topic := strings.ToLower(strings.TrimSpace(args[0]))
	if topic == "package" && len(args) > 1 {
		if usage, ok := packageHelp[strings.ToLower(args[1])]; ok {
			fmt.Print("\nPACKAGE / " + strings.ToUpper(args[1]) + "\n\n  " + usage + "\n\n  See also: transos help package\n")
			return 0
		}
		fmt.Printf("Unknown package action %q. Try: transos help package\n", args[1])
		return 2
	}
	if topic == "package" {
		fmt.Print("\nPACKAGE COMMANDS\n")
		for _, name := range []string{"seal", "verify", "inspect", "target-check", "execution-plan", "stage", "apply", "verify-applied", "restore"} {
			fmt.Printf("  %s\n", packageHelp[name])
		}
		fmt.Print("\nUse: transos help package <action>\n")
		return 0
	}
	for _, t := range helpTopics {
		if t.name == topic {
			fmt.Printf("\n%s — %s\n\nUsage:\n  %s\n\n%s\n\nNext: %s\n\n", strings.ToUpper(t.name), t.summary, t.usage, t.details, t.next)
			return 0
		}
	}
	fmt.Printf("Unknown help topic %q. Use 'transos help' to list categories.\n", topic)
	return 2
}

func printCategorizedHelp() {
	fmt.Print("\nTRANSOS  /  COMMAND GUIDE\n")
	fmt.Print("  Interactive: transos          Direct: transos <command> [flags]\n")
	fmt.Print("  Help:        transos help <command>  or  transos <command> --help\n")
	groups := []struct {
		title string
		names []string
	}{
		{"01  SOURCE & SETTINGS", []string{"config", "extract", "validate", "preview"}},
		{"02  MIGRATION WORKFLOW", []string{"plan", "preflight", "inject", "run-all"}},
		{"03  LINUX DEPLOYMENT", []string{"package"}},
		{"04  RECOVERY & INSPECTION", []string{"rollback", "status", "outputs"}},
		{"05  UTILITIES", []string{"version"}},
	}
	for _, g := range groups {
		fmt.Printf("\n%s\n", g.title)
		for _, name := range g.names {
			for _, t := range helpTopics {
				if t.name == name {
					fmt.Printf("  %-12s %s\n", t.name, t.summary)
					break
				}
			}
		}
	}
	fmt.Print("\nSuggested flow: config → extract → validate → plan → preflight → inject → package seal → package verify\n")
	fmt.Print("Linux target: package target-check → execution-plan → stage → apply → verify-applied\n")
	fmt.Print("Recovery: package restore (Linux) | rollback (generated source output)\n")
	fmt.Print("\nType 'help package' or 'help plan' for more detail. No installation is automatic.\n")
}
