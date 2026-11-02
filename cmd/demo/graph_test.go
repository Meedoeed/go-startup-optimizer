package main

import (
	"context"
	"strings"
	"testing"
)

func noopWork(context.Context) error {
	return nil
}

// 1. Проверяем сортировку зависимостей.
func TestOrderStepsUnordered(t *testing.T) {
	steps := []StartupStep{
		{
			Name:      "cache",
			DependsOn: []string{"database"},
			Work:      noopWork,
		},
		{
			Name:      "database",
			DependsOn: []string{"config"},
			Work:      noopWork,
		},
		{
			Name: "config",
			Work: noopWork,
		},
	}

	ordered, err := orderSteps(steps)

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"config",
		"database",
		"cache",
	}

	for i, name := range expected {
		if ordered[i].Name != name {
			t.Fatalf(
				"position %d: expected %s, got %s",
				i, name, ordered[i].Name,
			)
		}
	}
}

// 2. Проверяем неизвестную зависимость.
func TestOrderStepsUnknownDependency(t *testing.T) {
	steps := []StartupStep{
		{
			Name:      "cache",
			DependsOn: []string{"missing"},
			Work:      noopWork,
		},
	}

	_, err := orderSteps(steps)

	if err == nil ||
		!strings.Contains(err.Error(), "unknown component") {
		t.Fatalf(
			"expected unknown dependency error, got %v",
			err,
		)
	}
}

// 3. Проверяем одинаковые названия.
func TestOrderStepsDuplicateName(t *testing.T) {
	steps := []StartupStep{
		{Name: "config", Work: noopWork},
		{Name: "config", Work: noopWork},
	}

	_, err := orderSteps(steps)

	if err == nil ||
		!strings.Contains(err.Error(), "duplicate component") {
		t.Fatalf(
			"expected duplicate component error, got %v",
			err,
		)
	}
}

// 4. Проверяем циклическую зависимость.
func TestOrderStepsCycle(t *testing.T) {
	steps := []StartupStep{
		{
			Name:      "a",
			DependsOn: []string{"b"},
			Work:      noopWork,
		},
		{
			Name:      "b",
			DependsOn: []string{"c"},
			Work:      noopWork,
		},
		{
			Name:      "c",
			DependsOn: []string{"a"},
			Work:      noopWork,
		},
	}

	_, err := orderSteps(steps)

	if err == nil ||
		!strings.Contains(err.Error(), "cycle") {
		t.Fatalf(
			"expected cycle error, got %v",
			err,
		)
	}
}

// 5. Проверяем, что при ошибке графа
// никакой компонент не запускается.
func TestRunStartupRejectsInvalidGraphBeforeWork(
	t *testing.T,
) {
	called := false

	steps := []StartupStep{
		{
			Name: "config",
			Work: func(context.Context) error {
				called = true
				return nil
			},
		},
		{
			Name:      "cache",
			DependsOn: []string{"missing"},
			Work:      noopWork,
		},
	}

	_, err := runStartup(
		context.Background(),
		steps,
	)

	if err == nil {
		t.Fatal("expected invalid graph error")
	}

	if called {
		t.Fatal(
			"work must not run when graph is invalid",
		)
	}
}
