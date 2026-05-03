package catalog

// Manifest is the parsed `lab.yaml`. See docs/lab-schema.md.
type Manifest struct {
	SchemaVersion    int            `yaml:"schema_version"`
	ID               string         `yaml:"id"`
	Version          int            `yaml:"version"`
	Exam             string         `yaml:"exam"`
	ExamObjective    string         `yaml:"exam_objective"`
	Title            string         `yaml:"title"`
	Summary          string         `yaml:"summary"`
	Difficulty       string         `yaml:"difficulty"`
	EstimatedMinutes int            `yaml:"estimated_minutes"`
	Infrastructure   Infrastructure `yaml:"infrastructure"`
	Access           Access         `yaml:"access"`
	Instructions     string         `yaml:"instructions"`
	Tasks            []Task         `yaml:"tasks"`

	// Dir is the absolute filesystem path to the lab directory containing
	// `lab.yaml`. Populated by the loader, not present in the YAML.
	Dir string `yaml:"-"`
}

type Infrastructure struct {
	Provider   string         `yaml:"provider"` // aws | gcp | azure
	Module     string         `yaml:"module"`   // path relative to lab dir
	TTLMinutes int            `yaml:"ttl_minutes"`
	Inputs     map[string]any `yaml:"inputs"`
}

type Access struct {
	Kind   string `yaml:"kind"`   // kubeconfig | ssh
	Output string `yaml:"output"` // name of the Terraform output to read
}

type Task struct {
	ID           string       `yaml:"id"`
	Title        string       `yaml:"title"`
	Instructions string       `yaml:"instructions"`
	Validations  []Validation `yaml:"validations"`
}

type Validation struct {
	Kind           string   `yaml:"kind"` // kubectl | shell | http
	Args           []string `yaml:"args,omitempty"`
	Script         string   `yaml:"script,omitempty"`
	URL            string   `yaml:"url,omitempty"`
	ExpectEquals   *string  `yaml:"expect_equals,omitempty"`
	ExpectContains *string  `yaml:"expect_contains,omitempty"`
	ExpectExitCode *int     `yaml:"expect_exit_code,omitempty"`
	ExpectStatus   *int     `yaml:"expect_status,omitempty"`
	ExpectBodyHas  *string  `yaml:"expect_body_contains,omitempty"`
}
