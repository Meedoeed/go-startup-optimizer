package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunStepSuccess(t *testing.T) {
	step := StartupStep{
		Name: "config",
		Work: func(ctx context.Context) error {
			return nil
		},
	}

	result := runStep(context.Background(), step)

	if result.Status != StatusOK || result.Err != nil {
		t.Fatalf("expected success, got %+v", result)
	}
}

func TestRunStepTimeout(t *testing.T) {
	step := StartupStep{
		Name:    "database",
		Timeout: 20 * time.Millisecond,
		Work:    fakeWork(time.Second, nil),
	}

	result := runStep(context.Background(), step)

	if result.Status != StatusFailed {
		t.Fatalf(
			"expected failed status, got %s",
			result.Status,
		)
	}

	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Fatalf(
			"expected deadline exceeded, got %v",
			result.Err,
		)
	}
}

func TestRunStartupBlocksDependents(t *testing.T) {
	dbErr := errors.New("connection refused")

	cacheCalled := false
	workerCalled := false

	steps := []StartupStep{
		{
			Name: "config",
			Work: fakeWork(0, nil),
		},
		{
			Name:      "database",
			DependsOn: []string{"config"},
			Work: func(ctx context.Context) error {
				return dbErr
			},
		},
		{
			Name:      "cache",
			DependsOn: []string{"database"},
			Work: func(ctx context.Context) error {
				cacheCalled = true
				return nil
			},
		},
		{
			Name:      "metrics-worker",
			DependsOn: []string{"config"},
			Work: func(ctx context.Context) error {
				workerCalled = true
				return nil
			},
		},
	}

	report, err := runStartup(
		context.Background(),
		steps,
	)

	if err != nil {
		t.Fatalf("unexpected graph error: %v", err)
	}

	if report.Succeeded {
		t.Fatal("expected failed startup")
	}

	if got := report.Results[1]; got.Status != StatusFailed ||
		!errors.Is(got.Err, dbErr) {
		t.Fatalf("database: unexpected result %+v", got)
	}

	if got := report.Results[2]; got.Status != StatusBlocked ||
		got.BlockedBy != "database" {
		t.Fatalf("cache: unexpected result %+v", got)
	}

	if cacheCalled || !workerCalled {
		t.Fatalf(
			"cacheCalled=%v workerCalled=%v",
			cacheCalled,
			workerCalled,
		)
	}

	if report.Results[3].Status != StatusOK {
		t.Fatal("independent worker should be ok")
	}
}

func TestRunStepCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	cancel()

	result := runStep(ctx, StartupStep{
		Name: "job",
		Work: fakeWork(time.Hour, nil),
	})

	if result.Status != StatusFailed ||
		!errors.Is(result.Err, context.Canceled) {
		t.Fatalf(
			"expected cancellation, got %+v",
			result,
		)
	}
}
