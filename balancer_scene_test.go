package superspatial

import (
	"fmt"
	"testing"
)

func TestCalcRequiredWorkers(t *testing.T) {
	var tests = []struct {
		clients int
		workers int
	}{
		{0, 1},
		{1, 1},
		{2, 1},
		{3, 1},
		{4, 4},
		{7, 4},
		{8, 9},
		{31, 9},
		{32, 16},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d clients should be %d workers", tt.clients, tt.workers), func(t *testing.T) {
			log.Printf("Test: %+v", t)

			nw := calcRequiredWorkers(tt.clients)
			if nw != tt.workers {
				t.Errorf("got %d, want %d", nw, tt.workers)
			}

		})
	}

}
