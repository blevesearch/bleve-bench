package main

import "testing"

func TestAvgStddev(t *testing.T) {
	tests := []struct {
		inputs []float64
		avg    float64
		stddev float64
	}{
		{
			inputs: []float64{2, 4, 4, 4, 5, 5, 7, 9},
			avg:    5.0,
			stddev: 2.0,
		},
	}

	for _, test := range tests {
		avg := average(test.inputs)
		if avg != test.avg {
			t.Errorf("expected avg: %f got %f for inputs: %v", test.avg, avg, test.inputs)
		}
		stddev := stddev(test.inputs, avg)
		if stddev != test.stddev {
			t.Errorf("expected stddev: %f got %f for inputs: %v", test.stddev, stddev, test.inputs)
		}
	}
}
