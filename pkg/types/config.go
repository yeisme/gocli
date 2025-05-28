package types

type (
	Config struct {
		Name    string    `yaml:"name" json:"name"`
		Version string    `yaml:"version" json:"version"`
		Build   []Command `yaml:"build" json:"build"`
		Run     []Command `yaml:"run" json:"run"`
		Dev     []Command `yaml:"dev" json:"dev"`
		Clean   []Command `yaml:"clean" json:"clean"`
		Lint    []Command `yaml:"lint" json:"lint"`
		Help    []Command `yaml:"help" json:"help"`
		Release []Command `yaml:"release" json:"release"`
		Test    []Command `yaml:"test" json:"test"`
		Deps    []Command `yaml:"deps" json:"deps"`
		Project Project   `yaml:"project" json:"project"`
		Tools   Tools     `yaml:"tools" json:"tools"`
		Plugins Plugins   `yaml:"plugins" json:"plugins"`
	}

	Command struct {
		Name        string   `yaml:"name" json:"name"`
		Description string   `yaml:"description" json:"description"`
		Cmd         []string `yaml:"cmd" json:"cmd"`
	}

	Project struct {
		Name        string  `yaml:"name" json:"name"`
		Version     string  `yaml:"version" json:"version"`
		Description string  `yaml:"description" json:"description"`
		Author      string  `yaml:"author" json:"author"`
		License     string  `yaml:"license" json:"license"`
		GoVersion   string  `yaml:"go_version" json:"go_version"`
		Repository  string  `yaml:"repository" json:"repository"`
		Src         string  `yaml:"src" json:"src"`
		Manager     Manager `yaml:"manager" json:"manager"`
	}

	Manager struct {
		Make   []ManagerItem `yaml:"make,omitempty" json:"make,omitempty"`
		Tasks  []ManagerItem `yaml:"tasks,omitempty" json:"tasks,omitempty"`
		Just   []ManagerItem `yaml:"just,omitempty" json:"just,omitempty"`
		VSCode []ManagerItem `yaml:"vscode,omitempty" json:"vscode,omitempty"`
	}

	ManagerItem struct {
		Name string `yaml:"name" json:"name"`
		Dir  string `yaml:"dir" json:"dir"`
	}

	Tools struct {
		Dev    []DevTool    `yaml:"dev" json:"dev"`
		Go     []GoTool     `yaml:"go" json:"go"`
		Git    []GitTool    `yaml:"git" json:"git"`
		Custom []CustomTool `yaml:"custom,omitempty" json:"custom,omitempty"`
	}

	DevTool struct {
		Name    string `yaml:"name" json:"name"`
		Version string `yaml:"version" json:"version"`
	}

	GoTool struct {
		Name  string   `yaml:"name" json:"name"`
		URL   string   `yaml:"url" json:"url"`
		Bin   string   `yaml:"bin" json:"bin"`
		Flags []string `yaml:"flags,omitempty" json:"flags,omitempty"`
	}

	GitTool struct {
		Name        string `yaml:"name" json:"name"`
		Description string `yaml:"description" json:"description"`
		URL         string `yaml:"url" json:"url"`
		Type        string `yaml:"type" json:"type"`
		Recipe      string `yaml:"recipe" json:"recipe"`
	}

	CustomTool struct {
		Name  string   `yaml:"name" json:"name"`
		Cmd   string   `yaml:"cmd" json:"cmd"`
		Needs []string `yaml:"needs,omitempty" json:"needs,omitempty"`
	}

	Plugins struct {
		Enable bool   `yaml:"enable" json:"enable"`
		Dir    string `yaml:"dir" json:"dir"`
	}
)
