package deployment

import (
	"fmt"
	"runtime"
)

// PlannedAction is an advisory description, not an executable instruction.
type PlannedAction struct {
	Artifact         string
	Action           string
	RequiresApproval bool
	Privileged       bool
}

type ExecutionPlan struct {
	HostOS        string
	VerifiedFiles int
	Actions       []PlannedAction
	Warnings      []string
	Blockers      []string
}

// PlanExecution verifies the bundle and describes proposed actions without
// executing commands, writing files, or elevating privileges.
func PlanExecution(dir string) (ExecutionPlan, error) {
	result := ExecutionPlan{HostOS: runtime.GOOS}
	v, err := Verify(dir)
	if err != nil {
		return result, fmt.Errorf("verify deployment bundle: %w", err)
	}
	if !v.Valid() {
		return result, fmt.Errorf("deployment bundle integrity failed: %v", v.Problems)
	}
	result.VerifiedFiles = v.FilesChecked
	result.Actions = []PlannedAction{
		{Artifact: "transos_env.conf", Action: "Review environment exports before sourcing", RequiresApproval: true},
		{Artifact: ".bashrc", Action: "Review optional Bash integration; do not overwrite existing dotfiles", RequiresApproval: true},
		{Artifact: ".zshrc", Action: "Review optional Zsh integration; do not overwrite existing dotfiles", RequiresApproval: true},
		{Artifact: "install_dependencies.sh", Action: "Review package commands; installation may modify the system", RequiresApproval: true, Privileged: true},
	}
	result.Warnings = append(result.Warnings, "package checksums establish integrity against the local manifest, not publisher authenticity")
	result.Warnings = append(result.Warnings, "generated shell files must be reviewed before execution; no installation is authorized")
	if runtime.GOOS != "linux" {
		result.Blockers = append(result.Blockers, "execution is Linux-only; this is a planning preview")
	} else {
		target := DetectTarget()
		result.Blockers = append(result.Blockers, target.Blockers...)
		result.Warnings = append(result.Warnings, target.Warnings...)
	}
	return result, nil
}
