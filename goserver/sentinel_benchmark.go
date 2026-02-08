package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// BenchmarkCase represents one red-team sample.
type BenchmarkCase struct {
	Name        string `json:"name"`
	Action      string `json:"action"`
	Prompt      string `json:"prompt"`
	ExpectBlock bool   `json:"expect_block"`
}

// BenchmarkReport summarizes model behavior for judging/demo.
type BenchmarkReport struct {
	Total         int     `json:"total"`
	Correct       int     `json:"correct"`
	Accuracy      float64 `json:"accuracy"`
	TruePositive  int     `json:"true_positive"`
	FalsePositive int     `json:"false_positive"`
	TrueNegative  int     `json:"true_negative"`
	FalseNegative int     `json:"false_negative"`
}

func RunSentinelBenchmark(path string, guard *SentinelGuard) error {
	if guard == nil {
		return fmt.Errorf("sentinel guard is not configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cases []BenchmarkCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return err
	}

	report := BenchmarkReport{Total: len(cases)}

	for _, c := range cases {
		eval := guard.Evaluate(c.Action, c.Prompt)
		pred := eval.ShouldBlock
		ok := pred == c.ExpectBlock
		if ok {
			report.Correct++
		}

		switch {
		case c.ExpectBlock && pred:
			report.TruePositive++
		case !c.ExpectBlock && pred:
			report.FalsePositive++
		case !c.ExpectBlock && !pred:
			report.TrueNegative++
		case c.ExpectBlock && !pred:
			report.FalseNegative++
		}

		fmt.Printf("[%s] action=%s score=%d block=%v expect=%v tags=%v\n",
			c.Name, c.Action, eval.Score, pred, c.ExpectBlock, eval.Tags)
	}

	if report.Total > 0 {
		report.Accuracy = float64(report.Correct) / float64(report.Total)
	}

	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println("\nSentinel benchmark report:")
	fmt.Println(string(b))
	return nil
}
