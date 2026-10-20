package cli

import "fmt"

// migrationNextStep uses only the evidence already collected for the session.
// It deliberately does not infer plan approval, bundle integrity, or Linux readiness.
func migrationNextStep(state sessionState) string {
	if !state.profileReady {
		return "Capture source state: transos extract. Review configuration first with transos config."
	}
	if !state.profileValid {
		return "Review the source profile: transos validate, then transos preview. No deployment is implied."
	}
	if !state.outputReady {
		return "Review the migration plan (transos plan show) and run preflight. Generate only after review: transos inject."
	}
	return "Generated output exists, but integrity is unknown. Review the plan and run transos package seal, then transos package verify. On Linux, run target-check before deployment."
}

func printMigrationGuide(state sessionState) {
	uiHeader("MIGRATION GUIDE", "Read-only guidance. No commands are executed.")
	uiSection("Current evidence")
	profile := "MISSING"
	if state.profileValid {
		profile = "PARSED (not proof of plan approval)"
	} else if state.profileReady {
		profile = "PRESENT (needs validation)"
	}
	output := "MISSING"
	if state.outputReady {
		output = "PRESENT (integrity not checked)"
	}
	uiKeyValue("Profile", profile)
	uiKeyValue("Artifacts", output)
	uiSection("Recommended next step")
	for _, line := range uiWrap(migrationNextStep(state), uiWidth()-6) {
		fmt.Println("  " + line)
	}
	uiSection("Safe migration sequence")
	for _, step := range []string{
		"1. Configure and extract on Windows: config, extract, validate, preview.",
		"2. Review software decisions: plan show, then preflight.",
		"3. Generate and protect artifacts: inject, package seal, package verify.",
		"4. On Linux: package target-check and package execution-plan.",
		"5. Deploy only with explicit approval: package stage, package apply.",
		"6. Verify applied settings; use package restore for Linux recovery.",
	} {
		for _, line := range uiWrap(step, uiWidth()-6) {
			fmt.Println("  " + line)
		}
	}
	uiNote("Source-side rollback and Linux package restore are different operations. Package installation is never automatic.")
	uiFooter("Read command syntax: help plan | help package | help package apply")
}
