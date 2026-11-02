package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

func main() {
	steps := []StartupStep{
		{
			Name: "config",
			Work: fakeWork(
				100*time.Millisecond,
				nil,
			),
		},
		{
			Name:      "database",
			DependsOn: []string{"config"},
			Timeout:   450 * time.Millisecond,
			Work: fakeWork(
				1200*time.Millisecond,
				nil,
			),
		},
		{
			Name:      "cache",
			DependsOn: []string{"database"},
			Work: fakeWork(
				300*time.Millisecond,
				nil,
			),
		},
		{
			Name:      "http-server",
			DependsOn: []string{"cache"},
			Work: fakeWork(
				100*time.Millisecond,
				nil,
			),
		},
		{
			Name:      "background-worker",
			DependsOn: []string{"database"},
			Work: fakeWork(
				200*time.Millisecond,
				nil,
			),
		},
		{
			Name:      "metrics-worker",
			DependsOn: []string{"config"},
			Work: fakeWork(
				150*time.Millisecond,
				nil,
			),
		},
	}

	report, err := runStartup(
		context.Background(),
		steps,
	)

	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"INVALID STARTUP GRAPH:",
			err,
		)
		os.Exit(1)
	}

	fmt.Println("\nSTARTUP REPORT")

	for _, result := range report.Results {
		switch result.Status {
		case StatusOK:
			fmt.Printf(
				"[OK]      %-18s %s\n",
				result.Name,
				result.Duration.Round(time.Millisecond),
			)

		case StatusFailed:
			reason := "error"

			if errors.Is(
				result.Err,
				context.DeadlineExceeded,
			) {
				reason = "timeout"
			} else if errors.Is(
				result.Err,
				context.Canceled,
			) {
				reason = "canceled"
			}

			fmt.Printf(
				"[FAILED]  %-18s %s | %s: %v\n",
				result.Name,
				result.Duration.Round(time.Millisecond),
				reason,
				result.Err,
			)

		case StatusBlocked:
			fmt.Printf(
				"[BLOCKED] %-18s dependency: %s\n",
				result.Name,
				result.BlockedBy,
			)
		}
	}

	if !report.Succeeded {
		fmt.Printf(
			"\nSTARTUP FAILED | elapsed=%s\n",
			report.Duration.Round(time.Millisecond),
		)
		os.Exit(1)
	}

	fmt.Printf(
		"\nAPPLICATION READY | elapsed=%s\n",
		report.Duration.Round(time.Millisecond),
	)
}
