package marshaller

import "gopkg.in/yaml.v2"

// NewYaml creates a new instance of YAML marshaller.
func NewYaml() Service {
	return Yaml{}
}

// Yaml implements the YAML marshaller.
type Yaml struct {
}

// Marshal marshals the custom structure to the bytes sequence.
func (y Yaml) Marshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// Unmarshal unmarshalls the bytes sequence to the custom structure.
func (y Yaml) Unmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
