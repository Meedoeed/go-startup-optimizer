
package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type StartupStep struct {
	Name      string
	DependsOn []string
	Work      func() error
}

type StepStatus string

const (
	StatusOK      StepStatus = "ok"
	StatusFailed  StepStatus = "failed"
	StatusBlocked StepStatus = "blocked"
)

type StepResult struct {
	Name      string
	Status    StepStatus
	Duration  time.Duration
	Err       error
	BlockedBy string
}

func runStep(step StartupStep) StepResult {
	startedAt := time.Now()

	err := step.Work()

	result := StepResult{
		Name:     step.Name,
		Status:   StatusOK,
		Duration: time.Since(startedAt),
		Err:      err,
	}

	if err != nil {
		result.Status = StatusFailed
	}

	return result
}

func blockingDependency(
	step StartupStep,
	statuses map[string]StepStatus,
) string {
	for _, dependency := range step.DependsOn {
		status, exists := statuses[dependency]

		if !exists || status != StatusOK {
			return dependency
		}
	}

	return ""
}

func fakeWork(
	duration time.Duration,
	resultErr error,
) func() error {
	return func() error {
		time.Sleep(duration)
		return resultErr
	}
}

func main() {
	simulateDatabaseFailure := true

	var databaseErr error

	if simulateDatabaseFailure {
		databaseErr = errors.New("connection refused")
	}

	steps := []StartupStep{
		{
			Name: "config",
			Work: fakeWork(200*time.Millisecond, nil),
		},
		{
			Name:      "database",
			DependsOn: []string{"config"},
			Work:      fakeWork(1200*time.Millisecond, databaseErr),
		},
		{
			Name:      "cache",
			DependsOn: []string{"database"},
			Work:      fakeWork(700*time.Millisecond, nil),
		},
		{
			Name:      "http-server",
			DependsOn: []string{"cache"},
			Work:      fakeWork(100*time.Millisecond, nil),
		},
		{
			Name:      "background-worker",
			DependsOn: []string{"database"},
			Work:      fakeWork(500*time.Millisecond, nil),
		},
	}

	startedAt := time.Now()

	statuses := make(map[string]StepStatus)
	results := make([]StepResult, 0, len(steps))

	startupSucceeded := true

	for _, step := range steps {
		blockedBy := blockingDependency(step, statuses)

		if blockedBy != "" {
			statuses[step.Name] = StatusBlocked

			results = append(results, StepResult{
				Name:      step.Name,
				Status:    StatusBlocked,
				BlockedBy: blockedBy,
			})

			startupSucceeded = false
			continue
		}

		result := runStep(step)

		statuses[step.Name] = result.Status
		results = append(results, result)

		if result.Status != StatusOK {
			startupSucceeded = false
		}
	}

	fmt.Println("\nSTARTUP REPORT")

	for _, result := range results {
		switch result.Status {
		case StatusOK:
			fmt.Printf(
				"[OK]      %-12s %s\n",
				result.Name,
				result.Duration.Round(time.Millisecond),
			)

		case StatusFailed:
			fmt.Printf(
				"[FAILED]  %-12s %s | error: %v\n",
				result.Name,
				result.Duration.Round(time.Millisecond),
				result.Err,
			)

		case StatusBlocked:
			fmt.Printf(
				"[BLOCKED] %-12s dependency: %s\n",
				result.Name,
				result.BlockedBy,
			)
		}
	}

	if !startupSucceeded {
		fmt.Printf(
			"\nSTARTUP FAILED | elapsed=%s\n",
			time.Since(startedAt).Round(time.Millisecond),
		)

		os.Exit(1)
	}

	fmt.Printf(
		"\nAPPLICATION READY | elapsed=%s\n",
		time.Since(startedAt).Round(time.Millisecond),
	)
}
