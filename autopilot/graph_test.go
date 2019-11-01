package autopilot_test

import (
	"testing"

	"github.com/qtumproject/qtumsuite"
	"github.com/qtumproject/qlnd/autopilot"
)

// TestMedian tests the Median method.
func TestMedian(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		values []qtumsuite.Amount
		median qtumsuite.Amount
	}{
		{
			values: []qtumsuite.Amount{},
			median: 0,
		},
		{
			values: []qtumsuite.Amount{10},
			median: 10,
		},
		{
			values: []qtumsuite.Amount{10, 20},
			median: 15,
		},
		{
			values: []qtumsuite.Amount{10, 20, 30},
			median: 20,
		},
		{
			values: []qtumsuite.Amount{30, 10, 20},
			median: 20,
		},
		{
			values: []qtumsuite.Amount{10, 10, 10, 10, 5000000},
			median: 10,
		},
	}

	for _, test := range testCases {
		res := autopilot.Median(test.values)
		if res != test.median {
			t.Fatalf("expected median %v, got %v", test.median, res)
		}
	}
}
