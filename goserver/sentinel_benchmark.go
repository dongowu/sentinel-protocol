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
	Precision     float64 `json:"precision"`
	Recall        float64 `json:"recall"`
	F1            float64 `json:"f1"`
	BlockRate     float64 `json:"block_rate"`
}

func RunSentinelBenchmarkWithReport(path string, guard *SentinelGuard) (*BenchmarkReport, error) {
	if guard == nil {
		return nil, fmt.Errorf("sentinel guard is not configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cases []BenchmarkCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}

	report := BenchmarkReport{Total: len(cases)}
	blockedPredictions := 0

	for _, c := range cases {
		eval := guard.Evaluate(c.Action, c.Prompt)
		pred := eval.ShouldBlock
		if pred {
			blockedPredictions++
		}
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
		report.BlockRate = float64(blockedPredictions) / float64(report.Total)
	}
	if report.TruePositive+report.FalsePositive > 0 {
		report.Precision = float64(report.TruePositive) / float64(report.TruePositive+report.FalsePositive)
	}
	if report.TruePositive+report.FalseNegative > 0 {
		report.Recall = float64(report.TruePositive) / float64(report.TruePositive+report.FalseNegative)
	}
	if report.Precision+report.Recall > 0 {
		report.F1 = 2 * report.Precision * report.Recall / (report.Precision + report.Recall)
	}

	return &report, nil
}
