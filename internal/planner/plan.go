package planner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/transos/transos/internal/schema"
)

const (
	DefaultPath = "migration_plan.json"
	PlanVersion = "1.0.0"

	DecisionPending = "pending"
	DecisionInclude = "include"
	DecisionExclude = "exclude"
)

var linuxPackagePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+:_-]*$`)

// Plan keeps human review decisions separate from the canonical source profile.
type Plan struct {
	Version   string `json:"plan_version"`
	ProfileID string `json:"profile_id"`
	Profile   string `json:"profile_path,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Items     []Item `json:"items"`
}

// Item is a reviewed software candidate. Target-specific mapping is stored here,
// not in the platform-neutral migration profile.
type Item struct {
	Name            string `json:"name"`
	Version         string `json:"version,omitempty"`
	Publisher       string `json:"publisher,omitempty"`
	Architecture    string `json:"architecture,omitempty"`
	InstallLocation string `json:"install_location,omitempty"`
	Scope           string `json:"scope,omitempty"`
	Source          string `json:"source,omitempty"`
	Classification  string `json:"classification,omitempty"`
	Decision        string `json:"decision"`
	LinuxPackage    string `json:"linux_package,omitempty"`
	Manual          bool   `json:"manual,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

type Summary struct {
	Included int
	Pending  int
	Excluded int
}

// CreateFromProfile creates a new plan whose candidates all require an explicit
// user decision. Existing plans are protected unless overwrite is requested.
func CreateFromProfile(profile *schema.MigrationProfile, path string, overwrite bool) (*Plan, error) {
	if profile == nil {
		return nil, errors.New("migration profile is nil")
	}
	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("validate migration profile: %w", err)
	}
	if path == "" {
		path = DefaultPath
	}
	if _, err := os.Stat(path); err == nil && !overwrite {
		return nil, fmt.Errorf("migration plan %q already exists; use 'plan create --force' to rebuild it", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("check migration plan %q: %w", path, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	plan := &Plan{
		Version:   PlanVersion,
		ProfileID: profile.Metadata.ProfileID,
		Profile:   "migration_profile.json",
		CreatedAt: now,
		UpdatedAt: now,
		Items:     make([]Item, 0, len(profile.Software)),
	}
	if strings.TrimSpace(profile.Metadata.ProfileID) == "" {
		return nil, errors.New("migration profile metadata.profile_id is empty")
	}

	for _, software := range profile.Software {
		name := strings.TrimSpace(software.Name)
		if name == "" {
			continue
		}
		plan.Items = append(plan.Items, Item{
			Name:            name,
			Version:         strings.TrimSpace(software.Version),
			Publisher:       strings.TrimSpace(software.Publisher),
			Architecture:    strings.TrimSpace(software.Architecture),
			InstallLocation: strings.TrimSpace(software.InstallLocation),
			Scope:           strings.TrimSpace(software.Scope),
			Source:          strings.TrimSpace(software.Source),
			Classification:  strings.TrimSpace(software.Classification),
			Decision:        DecisionPending,
		})
	}

	if err := Save(path, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func Load(path string) (*Plan, error) {
	if path == "" {
		path = DefaultPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read migration plan %q: %w", path, err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse migration plan %q: %w", path, err)
	}
	if err := Validate(&plan); err != nil {
		return nil, fmt.Errorf("validate migration plan %q: %w", path, err)
	}
	return &plan, nil
}

func Save(path string, plan *Plan) error {
	if path == "" {
		path = DefaultPath
	}
	if plan == nil {
		return errors.New("migration plan is nil")
	}
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if plan.Version == "" {
		plan.Version = PlanVersion
	}
	if err := Validate(plan); err != nil {
		return fmt.Errorf("validate migration plan before save: %w", err)
	}

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize migration plan: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create migration plan directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".migration-plan-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary migration plan: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary migration plan: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary migration plan: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary migration plan: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace migration plan %q: %w", path, err)
	}
	return nil
}

func Validate(plan *Plan) error {
	if plan == nil {
		return errors.New("migration plan is nil")
	}
	if plan.Version != PlanVersion {
		return fmt.Errorf("unsupported plan version %q, expected %q", plan.Version, PlanVersion)
	}
	if strings.TrimSpace(plan.ProfileID) == "" {
		return errors.New("profile_id is required")
	}
	if strings.TrimSpace(plan.CreatedAt) == "" || strings.TrimSpace(plan.UpdatedAt) == "" {
		return errors.New("created_at and updated_at are required")
	}
	for index, item := range plan.Items {
		if strings.TrimSpace(item.Name) == "" {
			return fmt.Errorf("items[%d].name is required", index)
		}
		if !validDecision(item.Decision) {
			return fmt.Errorf("items[%d] has invalid decision %q", index, item.Decision)
		}
		if item.LinuxPackage != "" && !ValidLinuxPackage(item.LinuxPackage) {
			return fmt.Errorf("items[%d] has invalid Linux package name %q", index, item.LinuxPackage)
		}
	}
	return nil
}

func ValidLinuxPackage(name string) bool {
	return linuxPackagePattern.MatchString(name)
}

func SummaryOf(plan *Plan) Summary {
	var result Summary
	if plan == nil {
		return result
	}
	for _, item := range plan.Items {
		switch item.Decision {
		case DecisionInclude:
			result.Included++
		case DecisionExclude:
			result.Excluded++
		default:
			result.Pending++
		}
	}
	return result
}

func SetDecision(plan *Plan, name, decision string) (int, error) {
	if plan == nil {
		return 0, errors.New("migration plan is nil")
	}
	decision = strings.ToLower(strings.TrimSpace(decision))
	if !validDecision(decision) {
		return 0, fmt.Errorf("invalid decision %q", decision)
	}
	indices := matchingIndices(plan, name)
	if len(indices) == 0 {
		return 0, fmt.Errorf("software %q was not found in the migration plan", name)
	}
	for _, index := range indices {
		plan.Items[index].Decision = decision
	}
	return len(indices), nil
}

func MapPackage(plan *Plan, name, packageName string) (int, error) {
	if plan == nil {
		return 0, errors.New("migration plan is nil")
	}
	packageName = strings.TrimSpace(packageName)
	if !ValidLinuxPackage(packageName) {
		return 0, fmt.Errorf("invalid Linux package name %q; use letters, digits, dot, plus, colon, underscore, or hyphen", packageName)
	}
	indices := matchingIndices(plan, name)
	if len(indices) == 0 {
		return 0, fmt.Errorf("software %q was not found in the migration plan", name)
	}
	for _, index := range indices {
		plan.Items[index].LinuxPackage = packageName
	}
	return len(indices), nil
}

func AddManual(plan *Plan, name string) error {
	if plan == nil {
		return errors.New("migration plan is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("software name cannot be empty")
	}
	for _, item := range plan.Items {
		if strings.EqualFold(strings.TrimSpace(item.Name), name) {
			return fmt.Errorf("software %q already exists in the migration plan", name)
		}
	}
	plan.Items = append(plan.Items, Item{
		Name:           name,
		Classification: schema.ClassificationUnknown,
		Decision:       DecisionPending,
		Manual:         true,
	})
	return nil
}

func RemoveManual(plan *Plan, name string) error {
	if plan == nil {
		return errors.New("migration plan is nil")
	}
	indices := matchingIndices(plan, name)
	if len(indices) == 0 {
		return fmt.Errorf("software %q was not found in the migration plan", name)
	}
	for _, index := range indices {
		if !plan.Items[index].Manual {
			return fmt.Errorf("%q was discovered from the source profile; use 'plan exclude' rather than removing it", name)
		}
	}
	remove := make(map[int]bool, len(indices))
	for _, index := range indices {
		remove[index] = true
	}
	filtered := make([]Item, 0, len(plan.Items)-len(indices))
	for index, item := range plan.Items {
		if !remove[index] {
			filtered = append(filtered, item)
		}
	}
	plan.Items = filtered
	return nil
}

// ResolveForInjection returns only explicitly included software entries. Pending
// and excluded items are deliberately omitted from installation candidates.
func ResolveForInjection(profile *schema.MigrationProfile, plan *Plan) ([]schema.Software, map[string]string, Summary, error) {
	if profile == nil {
		return nil, nil, Summary{}, errors.New("migration profile is nil")
	}
	if plan == nil {
		return nil, nil, Summary{}, errors.New("migration plan is nil")
	}
	if profile.Metadata.ProfileID != plan.ProfileID {
		return nil, nil, Summary{}, fmt.Errorf("migration plan belongs to profile %q but loaded profile is %q; re-extract and run 'plan create --force' to rebuild review decisions", plan.ProfileID, profile.Metadata.ProfileID)
	}
	if err := Validate(plan); err != nil {
		return nil, nil, Summary{}, err
	}

	software := make([]schema.Software, 0, len(plan.Items))
	overrides := make(map[string]string)
	summary := SummaryOf(plan)
	for _, item := range plan.Items {
		if item.Decision != DecisionInclude {
			continue
		}
		classification := item.Classification
		if classification == "" {
			classification = schema.ClassificationUnknown
		}
		software = append(software, schema.Software{
			Name:            item.Name,
			Version:         item.Version,
			Publisher:       item.Publisher,
			Architecture:    item.Architecture,
			InstallLocation: item.InstallLocation,
			Scope:           item.Scope,
			Source:          item.Source,
			Classification:  classification,
		})
		if item.LinuxPackage != "" {
			overrides[item.Name] = item.LinuxPackage
		}
	}
	return software, overrides, summary, nil
}

func matchingIndices(plan *Plan, name string) []int {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	indices := make([]int, 0)
	for index, item := range plan.Items {
		if strings.EqualFold(strings.TrimSpace(item.Name), name) {
			indices = append(indices, index)
		}
	}
	return indices
}

func validDecision(decision string) bool {
	switch decision {
	case DecisionPending, DecisionInclude, DecisionExclude:
		return true
	default:
		return false
	}
}
