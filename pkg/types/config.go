package types

type (
	UserConfig struct {
		*Config
	}

	Config struct {
		Name    string    `json:"name"`
		Version string    `json:"version"`
		Build   []Command `json:"build"`
		Run     []Command `json:"run"`
		Dev     []Command `json:"dev"`
		Clean   []Command `json:"clean"`
		Lint    []Command `json:"lint"`
		Help    []Command `json:"help"`
		Release []Command `json:"release"`
		Test    []Command `json:"test"`
		Deps    []Command `json:"deps"`
		Project Project   `json:"project"`
		Tools   Tools     `json:"tools"`
		Plugins Plugins   `json:"plugins"`
	}

	Command struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Cmds         []string `json:"cmds"`
	}

	Project struct {
		Name        string  `json:"name"`
		Version     string  `json:"version"`
		Description string  `json:"description"`
		Author      string  `json:"author"`
		License     string  `json:"license"`
		GoVersion   string  `json:"go_version"`
		Repository  string  `json:"repository"`
		Src         string  `json:"src"`
		Manager     Manager `json:"manager"`
	}

	Manager struct {
		Make   []ManagerItem `json:"make,omitempty"`
		Task   []ManagerItem `json:"task,omitempty"`
		Just   []ManagerItem `json:"just,omitempty"`
		VSCode []ManagerItem `json:"vscode,omitempty"`
	}

	ManagerItem struct {
		Name string `json:"name"`
		Dir  string `json:"dir"`
	}

	Tools struct {
		Dev    []DevTool    `json:"dev"`
		Go     []GoTool     `json:"go"`
		Git    []GitTool    `json:"git"`
		Custom []CustomTool `json:"custom,omitempty"`
	}

	DevTool struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	GoTool struct {
		Name  string   `json:"name"`
		URL   string   `json:"url"`
		Bin   string   `json:"bin"`
		Flags []string `json:"flags,omitempty"`
	}

	GitTool struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		URL         string `json:"url"`
		Type        string `json:"type"`
		Recipe      string `json:"recipe,omitempty"`
		Path        string `json:"path"`
	}

	CustomTool struct {
		Name  string   `json:"name"`
		Cmds  string   `json:"cmds"`
		Needs []string `json:"needs,omitempty"`
	}

	Plugins struct {
		Enable bool   `json:"enable"`
		Dir    string `json:"dir"`
		Update bool   `json:"update"`
	}
)
