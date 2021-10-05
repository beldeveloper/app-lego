package model

// VariablesSources is a model that contains the data for filling the variables in the configuration.
type VariablesSources struct {
	Repository Repository
	Branch     Branch
	Deployment Deployment
	CustomData []byte
}
