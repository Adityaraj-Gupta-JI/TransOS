package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/transos/transos/internal/planner"
	"github.com/transos/transos/internal/safety"
	"github.com/transos/transos/internal/wal"
)

// PreflightReport is a read-only assessment of whether the current profile and
// review plan are ready for artifact generation.
type PreflightReport struct {
	ProfilePath              string
	ProfileID                string
	EnvironmentCount         int
	SensitiveEnvironment     []string
	SoftwareCount            int
	RegistryCount            int
	PlanPath                 string
	PlanPresent              bool
	Included                 int
	Pending                  int
	Excluded                 int
	IncludedExplicitlyMapped int
	OutputDirectory          string
	OutputDirectoryExists    bool
	CheckpointCount          int
	Warnings                 []string
	Blockers                 []string
}

func (r *PreflightReport) Ready() bool {
	return r != nil && len(r.Blockers) == 0
}

// AnalyzePreflight inspects profile/plan/output state without writing files or
// modifying checkpoint history.
func AnalyzePreflight(profilePath, planPath, outputDir string) (*PreflightReport, error) {
	if strings.TrimSpace(profilePath) == "" {
		profilePath = DefaultProfilePath
	}
	if strings.TrimSpace(planPath) == "" {
		planPath = planner.DefaultPath
	}
	if strings.TrimSpace(outputDir) == "" {
		outputDir = DefaultOutputDir
	}

	profile, err := LoadProfile(profilePath)
	if err != nil {
		return nil, err
	}

	report := &PreflightReport{
		ProfilePath:          profilePath,
		ProfileID:            profile.Metadata.ProfileID,
		EnvironmentCount:     len(profile.Environment),
		SoftwareCount:        len(profile.Software),
		RegistryCount:        len(profile.Registry),
		PlanPath:             planPath,
		OutputDirectory:      outputDir,
		SensitiveEnvironment: make([]string, 0),
		Warnings:             make([]string, 0),
		Blockers:             make([]string, 0),
	}

	for _, env := range profile.Environment {
		if safety.IsSensitiveEnvironmentName(env.Name) {
			report.SensitiveEnvironment = append(report.SensitiveEnvironment, env.Name)
		}
	}
	if len(report.SensitiveEnvironment) > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d sensitive-looking environment variable(s) are present by name; values will be omitted from generated Linux configuration", len(report.SensitiveEnvironment)))
	}

	if _, statErr := os.Stat(planPath); statErr == nil {
		report.PlanPresent = true
		plan, loadErr := planner.Load(planPath)
		if loadErr != nil {
			report.Blockers = append(report.Blockers, fmt.Sprintf("migration plan cannot be loaded: %v", loadErr))
		} else {
			summary := planner.SummaryOf(plan)
			report.Included = summary.Included
			report.Pending = summary.Pending
			report.Excluded = summary.Excluded
			_, _, _, resolveErr := planner.ResolveForInjection(profile, plan)
			for _, item := range plan.Items {
				if item.Decision == planner.DecisionInclude && strings.TrimSpace(item.LinuxPackage) != "" {
					report.IncludedExplicitlyMapped++
				}
			}
			if resolveErr != nil {
				report.Blockers = append(report.Blockers, resolveErr.Error())
			} else {
				if report.Pending > 0 {
					report.Warnings = append(report.Warnings, fmt.Sprintf("%d software candidate(s) remain pending and will not be installed", report.Pending))
				}
				if report.Included == 0 {
					report.Warnings = append(report.Warnings, "no software is approved for installation; only non-software migration artifacts will be generated")
				}
			}
		}
	} else if errors.Is(statErr, os.ErrNotExist) {
		report.Warnings = append(report.Warnings, "no migration plan exists; legacy injection mode will consider software from the full profile")
	} else {
		report.Blockers = append(report.Blockers, fmt.Sprintf("cannot inspect migration plan %q: %v", planPath, statErr))
	}

	if info, statErr := os.Stat(outputDir); statErr == nil {
		report.OutputDirectoryExists = info.IsDir()
		if !info.IsDir() {
			report.Blockers = append(report.Blockers, fmt.Sprintf("output path %q exists but is not a directory", outputDir))
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		report.Blockers = append(report.Blockers, fmt.Sprintf("cannot inspect output path %q: %v", outputDir, statErr))
	}

	checkpoints, checkpointErr := wal.ListCheckpoints(outputDir)
	if checkpointErr != nil {
		report.Blockers = append(report.Blockers, fmt.Sprintf("cannot read checkpoint history: %v", checkpointErr))
	} else {
		report.CheckpointCount = len(checkpoints)
	}

	return report, nil
}
