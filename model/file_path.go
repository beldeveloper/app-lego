package model

// FilePath wraps the string for defining the paths to files and directories.
type FilePath string

const (
	// RepositoriesDir defines the name of the directory that contains the repositories.
	RepositoriesDir FilePath = "repositories"
	// CustomDir defines the name of the directory that contains the custom files.
	CustomDir FilePath = "custom_files"
	// ConfigDir defines the name of the directory that contains the custom configuration.
	ConfigDir FilePath = "config"
	// BranchesDir defines the name of the directory that contains the temporary files related to the every branch.
	BranchesDir FilePath = "branches"
	// ScriptsDir defines the name of the directory that contains the scripts.
	ScriptsDir FilePath = "scripts"
)
