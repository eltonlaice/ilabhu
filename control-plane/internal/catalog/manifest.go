package catalog

// Manifest is the parsed `exam.yaml`. See docs/exam-schema.md.
type Manifest struct {
	SchemaVersion    int            `yaml:"schema_version"`
	ID               string         `yaml:"id"`
	Version          int            `yaml:"version"`
	Exam             string         `yaml:"exam"`
	Title            string         `yaml:"title"`
	Summary          string         `yaml:"summary"`
	Difficulty       string         `yaml:"difficulty"`
	EstimatedMinutes int            `yaml:"estimated_minutes"`
	TimeLimitMinutes int            `yaml:"time_limit_minutes"`
	PassingScore     int            `yaml:"passing_score"`
	Domains          []Domain       `yaml:"domains"`
	Infrastructure   Infrastructure `yaml:"infrastructure"`
	Access           Access         `yaml:"access"`
	Instructions     string         `yaml:"instructions"`
	Tasks            []Task         `yaml:"tasks"`

	// Dir is the absolute filesystem path to the exam directory containing
	// `exam.yaml`. Populated by the loader, not present in the YAML.
	Dir string `yaml:"-"`
}

// Domain represents one weighted exam objective area. The sum of weights
// across all domains should be 100.
type Domain struct {
	Name   string `yaml:"name" json:"name"`
	Weight int    `yaml:"weight" json:"weight"`
}

// Infrastructure describes how to provision the exam environment. An exam
// declares one or more providers (aws, gcp, azure, digitalocean, byo-hosts);
// the user picks one when they start a session.
type Infrastructure struct {
	TTLMinutes int                     `yaml:"ttl_minutes"`
	Providers  map[string]ProviderSpec `yaml:"providers"`
}

// ProviderSpec is a discriminated union covering both Terraform-driven cloud
// providers (Module + Inputs) and the BYO-hosts adapter (SetupScript +
// TeardownScript + Roles). A loaded manifest is expected to populate exactly
// one shape; the active provider is selected at session-start time.
type ProviderSpec struct {
	// Cloud (Terraform).
	Module string         `yaml:"module,omitempty"`
	Inputs map[string]any `yaml:"inputs,omitempty"`

	// BYO-hosts.
	SetupScript    string     `yaml:"setup_script,omitempty"`
	TeardownScript string     `yaml:"teardown_script,omitempty"`
	Roles          []HostRole `yaml:"roles,omitempty"`
}

// HostRole describes a class of host the BYO-hosts user must supply, and the
// minimum hardware shape ilabhu expects per host in that role.
type HostRole struct {
	Name     string `yaml:"name"`
	Count    int    `yaml:"count"`
	MinSpecs Specs  `yaml:"min_specs"`
}

type Specs struct {
	CPU    int    `yaml:"cpu"`
	RAMGB  int    `yaml:"ram_gb"`
	Distro string `yaml:"distro,omitempty"`
}

type Access struct {
	Kind   string `yaml:"kind"`   // kubeconfig | ssh
	Output string `yaml:"output"` // name of the Terraform output to read
}

type Task struct {
	ID           string       `yaml:"id"`
	Title        string       `yaml:"title"`
	Domain       string       `yaml:"domain,omitempty"`
	Weight       int          `yaml:"weight,omitempty"`
	Instructions string       `yaml:"instructions"`
	Validations  []Validation `yaml:"validations"`
}

// Validation is one assertion inside a task. The discriminator is `kind`.
type Validation struct {
	Kind           string   `yaml:"kind"` // kubectl | shell | http
	Args           []string `yaml:"args,omitempty"`
	Script         string   `yaml:"script,omitempty"`
	URL            string   `yaml:"url,omitempty"`
	ExpectEquals   *string  `yaml:"expect_equals,omitempty"`
	ExpectContains *string  `yaml:"expect_contains,omitempty"`
	ExpectRegex    *string  `yaml:"expect_regex,omitempty"`
	ExpectExitCode *int     `yaml:"expect_exit_code,omitempty"`
	ExpectStatus   *int     `yaml:"expect_status,omitempty"`
	ExpectBodyHas  *string  `yaml:"expect_body_contains,omitempty"`
}
