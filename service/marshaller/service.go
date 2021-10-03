package marshaller

// Service defines the interface that is in charge of marshalling and unmarshalling the data.
type Service interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}
