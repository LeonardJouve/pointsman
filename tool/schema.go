package tool

import "github.com/google/jsonschema-go/jsonschema"

type Schema struct {
	Name        string
	Description string
	Parameters  *jsonschema.Schema
}

func New(name, description string, parameters *jsonschema.Schema) Schema {
	return Schema{Name: name, Description: description, Parameters: parameters}
}
