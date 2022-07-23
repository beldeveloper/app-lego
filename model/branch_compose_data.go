package model

// BranchComposeData stores data required for deployment and docker compose.
type BranchComposeData struct {
	PreDeploy       []Cmd                           `yaml:"pre_deploy"`
	PostDeploy      []Cmd                           `yaml:"post_deploy"`
	ComposeServices map[string]DockerComposeService `yaml:"compose"`
}
