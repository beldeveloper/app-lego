package model

// Secret is a model that stores the secret data.
type Secret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
