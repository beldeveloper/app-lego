package model

// FilePath wraps the string for defining the paths to files and directories.
type FilePath string

const (
	// RepositoriesDir defines the name of the directory that contains the repositories.
	RepositoriesDir FilePath = "repositories"
	// CustomDir defines the name of the directory that contains the custom files.
	CustomDir FilePath = "custom_files"
)
