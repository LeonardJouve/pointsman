package tool

import "errors"

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

	sleep, err := Sleep()
	if err != nil {
		return nil, err
	}

	return []Tool{read, write, sleep}, nil
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

	return Tool{}, errors.New("unknown tool: " + name)
}
