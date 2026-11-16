package tool

import "fmt"

// All returns every built-in tool.
func All() ([]Tool, error) {
	var err error = nil

	read, err := Read()
	if err != nil {
		return nil, err
	}

	write, err := Write()
	if err != nil {
		return nil, err
	}

	subagent, err := Subagent()
	if err != nil {
		return nil, err
	}

	bash, err := Bash()
	if err != nil {
		return nil, err
	}

	return []Tool{read, write, subagent, bash}, nil
}

// Get returns the built-in tool with the given name, and an error for an unknown name.
func Get(name string) (Tool, error) {
	tools, err := All()
	if err != nil {
		return Tool{}, err
	}

	for _, t := range tools {
		if t.Schema.Name == name {
			return t, nil
		}
	}

	return Tool{}, fmt.Errorf("unknown tool: %s", name)
}
