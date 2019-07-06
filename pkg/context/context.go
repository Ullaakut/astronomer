package context

// Context represents the context of the Astronomer scan.
type Context struct {
	RepoOwner          string
	RepoName           string
	GithubToken        string
	CacheDirectoryPath string

	// ScanAll makes astronomer scan every stargazer
	// when set to true.
	ScanAll bool

	// Amount of stars to scan in fastMode.
	Stars uint

	// Verbose enables the verbose mode.
	Verbose bool
}
