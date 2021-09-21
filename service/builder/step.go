package builder

type buildingStep struct {
	name   string
	action func() error
	next   *buildingStep
}
