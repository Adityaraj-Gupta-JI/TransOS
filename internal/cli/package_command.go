package cli

import (
	"fmt"
	"github.com/transos/transos/internal/deployment"
	"github.com/transos/transos/internal/injector"
	"strings"
)

func runPackage(args []string) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Println("Usage: transos package <seal|verify|inspect|target-check|execution-plan> [output_dir]")
		return 2
	}
	dir := injector.DefaultOutputDir
	if len(args) == 2 {
		dir = strings.TrimSpace(args[1])
		if dir == "" {
			return 2
		}
	}
	switch strings.ToLower(args[0]) {
	case "seal":
		m, err := deployment.Seal(dir)
		if err != nil {
			fmt.Printf("[-] Package sealing failed: %v\n", err)
			return 1
		}
		fmt.Printf("[+] Sealed %d artifacts in %s/%s\n", len(m.Files), dir, deployment.ManifestName)
		fmt.Println("    No installation scripts were executed.")
		return 0
	case "verify":
		v, err := deployment.Verify(dir)
		if err != nil {
			fmt.Printf("[-] Verification failed: %v\n", err)
			return 1
		}
		for _, p := range v.Problems {
			fmt.Println("[-]", p)
		}
		if !v.Valid() {
			return 1
		}
		fmt.Printf("[+] Verified %d artifacts; hashes match. No files changed.\n", v.FilesChecked)
		return 0
	case "target-check":
		r, err := deployment.CheckTarget(dir)
		if err != nil {
			fmt.Printf("[-] Target check blocked: %v\n", err)
			return 1
		}
		fmt.Printf("Target: %s/%s | distribution: %s %s | package manager: %s | shell: %s\n", r.OS, r.Architecture, r.Distro, r.DistroVersion, r.PackageManager, r.Shell)
		for _, warning := range r.Warnings {
			fmt.Println("[!]", warning)
		}
		for _, blocker := range r.Blockers {
			fmt.Println("[-]", blocker)
		}
		if !r.Ready() {
			return 1
		}
		fmt.Println("[+] Target compatibility checks passed. No installation was performed.")
		return 0
	case "execution-plan":
		plan, err := deployment.PlanExecution(dir)
		if err != nil {
			fmt.Printf("[-] Execution planning blocked: %v\n", err)
			return 1
		}
		fmt.Printf("Execution preview | host: %s | verified artifacts: %d\n", plan.HostOS, plan.VerifiedFiles)
		for _, a := range plan.Actions {
			privilege := "user review"
			if a.Privileged {
				privilege = "may require administrator privileges"
			}
			fmt.Printf("  %-25s %s (%s; approval required)\n", a.Artifact, a.Action, privilege)
		}
		for _, w := range plan.Warnings {
			fmt.Println("[!]", w)
		}
		for _, b := range plan.Blockers {
			fmt.Println("[-]", b)
		}
		fmt.Println("No commands executed and no files changed.")
		if len(plan.Blockers) > 0 {
			return 1
		}
		return 0
	case "inspect":
		m, err := deployment.Load(dir)
		if err != nil {
			fmt.Printf("[-] Manifest unavailable: %v\n", err)
			return 1
		}
		fmt.Printf("Package manifest v%d | created %s\n", m.Version, m.CreatedAt)
		for _, f := range m.Files {
			fmt.Printf("  %-28s %8d bytes  %s\n", f.Path, f.Size, f.SHA256)
		}
		fmt.Println("Inspect does not verify file contents; run package verify.")
		return 0
	default:
		fmt.Println("Usage: transos package <seal|verify|inspect|target-check|execution-plan> [output_dir]")
		return 2
	}
}
