package hypertest

type Arguements struct {
	Cmdline  string // pytest -v, go test ./..., etc
	RepoPath string // Path to the repo, defaults to "."
}
