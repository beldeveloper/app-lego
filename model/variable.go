package model

const (
	// VariableTypeBuilding defines the type of variables that are defined by the building process.
	VariableTypeBuilding = iota
	// VariableTypeCustom defines the type of variables that are defined by the external configuration.
	VariableTypeCustom
	// VariableTypeSecret defines the type of variables that are defined by the repository secrets.
	VariableTypeSecret
)

// Variable is a model that represents a configuration variable.
type Variable struct {
	Type  int    `json:"-"`
	Name  string `json:"name"`
	Value string `json:"value"`
}
