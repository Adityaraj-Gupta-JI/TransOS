package cli

import (
	"fmt"
	"strings"
)

type workspaceAction struct{ label, command, explanation string }
type workspace struct {
	name, description string
	actions           []workspaceAction
}

var workspaces = []workspace{
	{"Source & Settings", "Configure and capture the Windows source", []workspaceAction{{"View settings", "config", "Inspect extraction preferences"}, {"Extract source", "extract", "Capture Windows environment and software"}, {"Validate profile", "validate", "Check profile structure"}, {"Preview profile", "preview", "Review captured state"}}},
	{"Migration Planning", "Review software decisions and readiness", []workspaceAction{{"Plan overview", "plan show", "Review current decisions"}, {"Create plan", "plan create", "Create a migration decision plan"}, {"Preflight checks", "preflight", "Read-only safety checks"}, {"Dry run", "inject --dry-run", "Preview package generation"}}},
	{"Package Generation", "Generate and validate portable artifacts", []workspaceAction{{"Generate artifacts", "inject", "Generate Linux configuration bundle"}, {"Seal bundle", "package seal", "Record artifact checksums"}, {"Verify bundle", "package verify", "Check integrity"}, {"Inspect bundle", "package inspect", "Read manifest contents"}}},
	{"Linux Deployment", "Review and apply on the target Linux machine", []workspaceAction{{"Target check", "package target-check", "Check Linux compatibility"}, {"Execution plan", "package execution-plan", "Read-only deployment preview"}, {"Stage help", "help package stage", "Learn staging syntax and approval"}, {"Apply help", "help package apply", "Learn safe application syntax"}}},
	{"Recovery & Verification", "Verify changes and understand recovery", []workspaceAction{{"Verify applied help", "help package verify-applied", "Inspect applied-state verification"}, {"Restore help", "help package restore", "Review Linux restore requirements"}, {"Rollback checkpoints", "rollback list", "List source-side checkpoints"}, {"Generated outputs", "outputs", "Inspect source-generated artifacts"}}},
	{"Diagnostics & Help", "Inspect status and command documentation", []workspaceAction{{"Current status", "status", "Show local migration state"}, {"Command guide", "help", "Browse all commands"}, {"Package guide", "help package", "Explore package actions"}, {"Version", "version", "Display installed version"}}},
}

var activeWorkspace = -1

func renderWorkspaceHome() {
	uiHeader("MIGRATION CONSOLE", "Windows source  >  portable bundle  >  Linux target")
	state := readSessionState()
	profile := "NOT FOUND"
	if state.profileValid {
		profile = "VALID"
	} else if state.profileReady {
		profile = "REVIEW"
	}
	output := "NOT GENERATED"
	if state.outputReady {
		output = "PRESENT / UNVERIFIED"
	}
	checkpoint := "NONE"
	if state.walReady {
		checkpoint = "AVAILABLE"
	}
	uiSection("CURRENT WORKSPACE")
	uiKeyValue("Source profile", profile)
	uiKeyValue("Generated output", output)
	uiKeyValue("Checkpoints", checkpoint)
	fmt.Println("  Note: file presence does not establish package integrity.")
	uiSection("WORKSPACES")
	for i, w := range workspaces {
		uiMenuItem(i+1, w.name, w.description)
	}
	uiFooter("[1-6] open workspace  |  help [topic]  |  status  |  exit")
	fmt.Println("  Direct CLI commands are accepted from every workspace.")
}

func renderWorkspace(index int) {
	w := workspaces[index]
	uiHeader(fmt.Sprintf("%02d / %s", index+1, strings.ToUpper(w.name)), w.description)
	uiSection("ACTIONS")
	for i, a := range w.actions {
		uiMenuItem(i+1, a.label, a.explanation)
	}
	uiFooter("[1-4] run action  |  back  |  home  |  help [topic]  |  exit")
	if index == 3 || index == 4 {
		fmt.Println("  Deployment and restore require explicit approval; read help first.")
	}
}

func handleWorkspaceInput(args []string) bool {
	if len(args) == 0 {
		return true
	}
	cmd := strings.ToLower(args[0])
	switch cmd {
	case "home", "menu":
		activeWorkspace = -1
		renderWorkspaceHome()
		return true
	case "back":
		if activeWorkspace >= 0 {
			activeWorkspace = -1
			renderWorkspaceHome()
		} else {
			fmt.Println("Already at the home menu.")
		}
		return true
	case "dashboard":
		if activeWorkspace >= 0 {
			renderWorkspace(activeWorkspace)
		} else {
			renderWorkspaceHome()
		}
		return true
	}
	if len(args) != 1 {
		return false
	}
	n := 0
	for _, r := range cmd {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return false
	}
	if activeWorkspace < 0 {
		if n > len(workspaces) {
			fmt.Printf("Choose a workspace from 1 to %d.\n", len(workspaces))
			return true
		}
		activeWorkspace = n - 1
		renderWorkspace(activeWorkspace)
		return true
	}
	actions := workspaces[activeWorkspace].actions
	if n > len(actions) {
		fmt.Printf("Choose an action from 1 to %d, or type 'back'.\n", len(actions))
		return true
	}
	action := actions[n-1]
	fmt.Printf("\nAction: %s\nCommand: transos %s\n", action.label, action.command)
	runInteractiveCommand(action.command)
	return true
}
