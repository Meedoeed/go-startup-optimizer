package main

import (
	"context"
	"fmt"
	"time"
)

type StartupStep struct {
	Name      string
	DependsOn []string
	Timeout   time.Duration
	Work      func(context.Context) error
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

type StartupReport struct {
	Results   []StepResult
	Duration  time.Duration
	Succeeded bool
}

func runStep(
	ctx context.Context,
	step StartupStep,
) StepResult {
	stepCtx := ctx

	if step.Timeout > 0 {
		var cancel context.CancelFunc

		stepCtx, cancel = context.WithTimeout(
			ctx,
			step.Timeout,
		)
		defer cancel()
	}

	startedAt := time.Now()

	err := step.Work(stepCtx)

	if err == nil {
		err = stepCtx.Err()
	}

	status := StatusOK

	if err != nil {
		status = StatusFailed
	}

	return StepResult{
		Name:     step.Name,
		Status:   status,
		Duration: time.Since(startedAt),
		Err:      err,
	}
}

func blockingDependency(
	step StartupStep,
	statuses map[string]StepStatus,
) string {
	for _, dependency := range step.DependsOn {
		if statuses[dependency] != StatusOK {
			return dependency
		}
	}

	return ""
}

func runStartup(
	ctx context.Context,
	steps []StartupStep,
) (StartupReport, error) {
	// Сначала проверяем граф зависимостей.
	ordered, err := orderSteps(steps)

	if err != nil {
		return StartupReport{}, fmt.Errorf(
			"invalid startup graph: %w",
			err,
		)
	}

	startedAt := time.Now()

	report := StartupReport{
		Results:   make([]StepResult, 0, len(steps)),
		Succeeded: true,
	}

	statuses := make(map[string]StepStatus)

	// Выполняем компоненты в безопасном порядке.
	for _, step := range ordered {
		dependency := blockingDependency(
			step,
			statuses,
		)

		if dependency != "" {
			statuses[step.Name] = StatusBlocked

			report.Results = append(
				report.Results,
				StepResult{
					Name:      step.Name,
					Status:    StatusBlocked,
					BlockedBy: dependency,
				},
			)

			report.Succeeded = false
			continue
		}

		result := runStep(ctx, step)

		statuses[step.Name] = result.Status

		report.Results = append(
			report.Results,
			result,
		)

		if result.Status != StatusOK {
			report.Succeeded = false
		}
	}

	report.Duration = time.Since(startedAt)

	return report, nil
}

func fakeWork(
	duration time.Duration,
	resultErr error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			return resultErr
		}
	}
}
