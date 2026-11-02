package main

import "fmt"

func orderSteps(steps []StartupStep) ([]StartupStep, error) {
	byName := make(map[string]StartupStep, len(steps))

	for _, step := range steps {
		if step.Name == "" {
			return nil, fmt.Errorf(
				"component name cannot be empty",
			)
		}

		if _, exists := byName[step.Name]; exists {
			return nil, fmt.Errorf(
				"duplicate component %q",
				step.Name,
			)
		}

		if step.Work == nil {
			return nil, fmt.Errorf(
				"component %q has no Work function",
				step.Name,
			)
		}

		byName[step.Name] = step
	}

	for _, step := range steps {
		for _, dependency := range step.DependsOn {
			if _, exists := byName[dependency]; !exists {
				return nil, fmt.Errorf(
					"component %q depends on unknown component %q",
					step.Name,
					dependency,
				)
			}
		}
	}

	state := make(map[string]uint8, len(steps))

	ordered := make([]StartupStep, 0, len(steps))

	var visit func(string) error

	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf(
				"dependency cycle detected at %q",
				name,
			)

		case 2:
			return nil
		}

		state[name] = 1

		for _, dependency := range byName[name].DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}

		state[name] = 2

		ordered = append(ordered, byName[name])

		return nil
	}

	for _, step := range steps {
		if err := visit(step.Name); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}
