package types

type (
	UserConfig struct {
		*Config
	}

	Config struct {
		Name    string    `json:"name"`
		Version string    `json:"version,omitempty"`
		Build   []Command `json:"build,omitempty"`
		Run     []Command `json:"run,omitempty"`
		Dev     []Command `json:"dev,omitempty"`
		Clean   []Command `json:"clean,omitempty"`
		Lint    []Command `json:"lint,omitempty"`
		Help    []Command `json:"help,omitempty"`
		Release []Command `json:"release,omitempty"`
		Test    []Command `json:"test,omitempty"`
		Deps    []Command `json:"deps,omitempty"`
		Project Project   `json:"project,omitempty"`
		Tools   Tools     `json:"tools,omitempty"`
		Plugins Plugins   `json:"plugins,omitempty"`
	}

	Command struct {
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Cmds        []string `json:"cmds"`
	}

	Project struct {
		Name        string  `json:"name"`
		Version     string  `json:"version,omitempty"`
		Description string  `json:"description,omitempty"`
		Author      string  `json:"author,omitempty"`
		License     string  `json:"license,omitempty"`
		GoVersion   string  `json:"go_version,omitempty"`
		Repository  string  `json:"repository,omitempty"`
		Src         string  `json:"src,omitempty"`
		Manager     Manager `json:"manager,omitempty"`
	}

	Manager struct {
		Make   []ManagerItem `json:"make,omitempty"`
		Task   []ManagerItem `json:"task,omitempty"`
		Just   []ManagerItem `json:"just,omitempty"`
		VSCode []ManagerItem `json:"vscode,omitempty"`
	}

	ManagerItem struct {
		Name string `json:"name"`
		Dir  string `json:"dir,omitempty"`
	}

	Tools struct {
		Dev    []DevTool    `json:"dev,omitempty"`
		Go     []GoTool     `json:"go,omitempty"`
		Git    []GitTool    `json:"git,omitempty"`
		Custom []CustomTool `json:"custom,omitempty"`
	}

	DevTool struct {
		Name    string `json:"name"`
		Version string `json:"version,omitempty"`
	}

	GoTool struct {
		Name  string   `json:"name"`
		URL   string   `json:"url,omitempty"`
		Bin   string   `json:"bin,omitempty"`
		Flags []string `json:"flags,omitempty"`
	}

	GitTool struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		URL         string `json:"url,omitempty"`
		Type        string `json:"type,omitempty"`
		Recipe      string `json:"recipe,omitempty"`
		Path        string `json:"path,omitempty"`
	}

	CustomTool struct {
		Name  string   `json:"name"`
		Cmds  string   `json:"cmds"`
		Needs []string `json:"needs,omitempty"`
	}

	Plugins struct {
		Enable bool   `json:"enable,omitempty"`
		Dir    string `json:"dir,omitempty"`
		Update bool   `json:"update,omitempty"`
	}
)
