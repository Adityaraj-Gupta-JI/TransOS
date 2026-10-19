package cli

import (
	"fmt"
	"strings"

	"github.com/transos/transos/internal/app"
	"github.com/transos/transos/internal/planner"
)

func runPlan(args []string) int {
	if len(args) == 0 {
		printPlanUsage()
		return 2
	}

	action := strings.ToLower(strings.TrimSpace(args[0]))
	operands := args[1:]

	switch action {
	case "create":
		force := false
		for _, operand := range operands {
			if operand == "--force" || operand == "-f" {
				force = true
			} else {
				fmt.Printf(Red+"[-] Unexpected argument for plan create: %s\n"+Reset, operand)
				return 2
			}
		}
		profile, err := app.LoadProfile(app.DefaultProfilePath)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot create plan: %v\n"+Reset, err)
			return 1
		}
		plan, err := planner.CreateFromProfile(profile, planner.DefaultPath, force)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot create plan: %v\n"+Reset, err)
			return 1
		}
		fmt.Printf(Green+"✓ Created %s with %d software candidates.\n"+Reset, planner.DefaultPath, len(plan.Items))
		fmt.Println("  All candidates start as pending. Review them before generating the Linux installer.")
		fmt.Println("  Next: plan show, then plan include/exclude/map/add as needed.")
		return 0

	case "show", "list":
		plan, err := planner.Load(planner.DefaultPath)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot load plan: %v\n"+Reset, err)
			return 1
		}
		summary := planner.SummaryOf(plan)
		fmt.Printf("Migration plan: %s\nProfile ID: %s\n", planner.DefaultPath, plan.ProfileID)
		fmt.Printf("Items: %d | Included: %d | Pending: %d | Excluded: %d\n\n", len(plan.Items), summary.Included, summary.Pending, summary.Excluded)
		fmt.Printf("%-10s %-34s %-16s %s\n", "DECISION", "SOFTWARE", "VERSION", "LINUX PACKAGE")
		fmt.Printf("%-10s %-34s %-16s %s\n", "----------", "--------", "-------", "-------------")
		for _, item := range plan.Items {
			name := item.Name
			if item.Manual {
				name += " [manual]"
			}
			fmt.Printf("%-10s %-34.34s %-16.16s %s\n", item.Decision, name, item.Version, item.LinuxPackage)
		}
		return 0

	case "include", "exclude":
		name := strings.TrimSpace(strings.Join(operands, " "))
		if name == "" {
			fmt.Printf(Red+"[-] Usage: plan %s <software name>\n"+Reset, action)
			return 2
		}
		plan, err := planner.Load(planner.DefaultPath)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot load plan: %v\n"+Reset, err)
			return 1
		}
		count, err := planner.SetDecision(plan, name, action)
		if err != nil {
			fmt.Printf(Red+"[-] %v\n"+Reset, err)
			return 1
		}
		if err := planner.Save(planner.DefaultPath, plan); err != nil {
			fmt.Printf(Red+"[-] Cannot save plan: %v\n"+Reset, err)
			return 1
		}
		fmt.Printf(Green+"✓ Marked %d matching item(s) as %s: %s\n"+Reset, count, action, name)
		return 0

	case "map":
		if len(operands) < 2 {
			fmt.Println(Red + "[-] Usage: plan map <software name> <linux-package>" + Reset)
			return 2
		}
		name := strings.TrimSpace(strings.Join(operands[:len(operands)-1], " "))
		packageName := strings.TrimSpace(operands[len(operands)-1])
		plan, err := planner.Load(planner.DefaultPath)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot load plan: %v\n"+Reset, err)
			return 1
		}
		count, err := planner.MapPackage(plan, name, packageName)
		if err != nil {
			fmt.Printf(Red+"[-] %v\n"+Reset, err)
			return 1
		}
		if err := planner.Save(planner.DefaultPath, plan); err != nil {
			fmt.Printf(Red+"[-] Cannot save plan: %v\n"+Reset, err)
			return 1
		}
		fmt.Printf(Green+"✓ Mapped %d matching item(s): %s → %s\n"+Reset, count, name, packageName)
		fmt.Println("  Mapping does not approve installation. Use 'plan include' separately.")
		return 0

	case "add":
		name := strings.TrimSpace(strings.Join(operands, " "))
		if name == "" {
			fmt.Println(Red + "[-] Usage: plan add <software name>" + Reset)
			return 2
		}
		plan, err := planner.Load(planner.DefaultPath)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot load plan: %v\n"+Reset, err)
			return 1
		}
		if err := planner.AddManual(plan, name); err != nil {
			fmt.Printf(Red+"[-] %v\n"+Reset, err)
			return 1
		}
		if err := planner.Save(planner.DefaultPath, plan); err != nil {
			fmt.Printf(Red+"[-] Cannot save plan: %v\n"+Reset, err)
			return 1
		}
		fmt.Printf(Green+"✓ Added manual candidate as pending: %s\n"+Reset, name)
		return 0

	case "remove":
		name := strings.TrimSpace(strings.Join(operands, " "))
		if name == "" {
			fmt.Println(Red + "[-] Usage: plan remove <manual software name>" + Reset)
			return 2
		}
		plan, err := planner.Load(planner.DefaultPath)
		if err != nil {
			fmt.Printf(Red+"[-] Cannot load plan: %v\n"+Reset, err)
			return 1
		}
		if err := planner.RemoveManual(plan, name); err != nil {
			fmt.Printf(Red+"[-] %v\n"+Reset, err)
			return 1
		}
		if err := planner.Save(planner.DefaultPath, plan); err != nil {
			fmt.Printf(Red+"[-] Cannot save plan: %v\n"+Reset, err)
			return 1
		}
		fmt.Printf(Green+"✓ Removed manually added candidate: %s\n"+Reset, name)
		return 0

	default:
		printPlanUsage()
		return 2
	}
}

func printPlanUsage() {
	fmt.Println("Migration plan commands:")
	fmt.Println("  transos plan create [--force]    Create pending review items from migration_profile.json")
	fmt.Println("  transos plan show                List decisions and Linux package mappings")
	fmt.Println("  transos plan include <name>      Approve software for target package generation")
	fmt.Println("  transos plan exclude <name>      Explicitly exclude software")
	fmt.Println("  transos plan map <name> <pkg>    Set an apt/dnf package name (approval is separate)")
	fmt.Println("  transos plan add <name>          Add a manual candidate as pending")
	fmt.Println("  transos plan remove <name>       Remove a manually added candidate")
}
