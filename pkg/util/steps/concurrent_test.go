package steps

import (
	"context"
	"fmt"
	"slices"
	"testing"
)

type teststructConcurrent struct{}

// functionnames that will be used in the conditionFunction below
// All the keys of map timeoutConditionErrors
func (n *teststructConcurrent) attachNSGs(context.Context) error              { return nil }
func (n *teststructConcurrent) apiServersReady(context.Context) error         { return nil }
func (n *teststructConcurrent) minimumWorkerNodesReady(context.Context) error { return nil }

func Test_concurrentStep_String(t *testing.T) {
	ts := teststructConcurrent{}
	tests := []struct {
		name  string // description of this test case
		steps []Step
		want  string
	}{
		{
			name:  "nosteps",
			steps: []Step{},
			want:  "[Action nosteps []]",
		},
		{
			name:  "nilsteps",
			steps: nil,
			want:  "[Action nilsteps []]",
		},
		{
			name:  "onestep",
			steps: []Step{Action(ts.attachNSGs)},
			want:  "[Action onestep [attachNSGs]]",
		},
		{
			name: "threesteps",
			steps: []Step{
				Action(ts.attachNSGs),
				Action(ts.apiServersReady),
				Action(ts.minimumWorkerNodesReady),
			},
			want: "[Action threesteps [attachNSGs, apiServersReady, minimumWorkerNodesReady]]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Concurrent(tt.name, tt.steps)
			got := s.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

type joinedErr interface {
	Unwrap() []error
}

func Test_concurrentStep_run(t *testing.T) {
	errOne := fmt.Errorf("one")
	errTwo := fmt.Errorf("two")
	errThree := fmt.Errorf("three")

	tests := []struct {
		name     string
		steps    []Step
		wantErr  bool
		wantErrs []error
	}{
		{
			name: "three errors",
			steps: []Step{
				Action(func(ctx context.Context) error { return errOne }),
				Action(func(ctx context.Context) error { return errTwo }),
				Action(func(ctx context.Context) error { return errThree }),
			},
			wantErr:  true,
			wantErrs: []error{errOne, errTwo, errThree},
		},
		{
			name: "Three runs, one err",
			steps: []Step{
				Action(func(ctx context.Context) error { return nil }),
				Action(func(ctx context.Context) error { return nil }),
				Action(func(ctx context.Context) error { return errThree }),
			},
			wantErr:  true,
			wantErrs: []error{errThree},
		},
		{
			name: "Three runs, no err",
			steps: []Step{
				Action(func(ctx context.Context) error { return nil }),
				Action(func(ctx context.Context) error { return nil }),
				Action(func(ctx context.Context) error { return nil }),
			},
			wantErrs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Concurrent("test", tt.steps)

			gotErr := s.run(context.Background(), nil)

			if gotErr != nil {
				je, ok := gotErr.(joinedErr)
				if !ok {
					t.Errorf("Unexpected error type: %v", gotErr)
					return
				}

				gotErrs := je.Unwrap()
				if len(tt.wantErrs) != len(gotErrs) {
					t.Errorf("run() = %v, want %v", gotErr, tt.wantErrs)
					return
				}

				for _, curErr := range gotErrs {
					if !slices.Contains(tt.wantErrs, curErr) {
						t.Errorf("run() = %v, want %v", gotErr, tt.wantErrs)
						return
					}
				}

				for _, curErr := range tt.wantErrs {
					if !slices.Contains(gotErrs, curErr) {
						t.Errorf("run() = %v, want %v", gotErr, tt.wantErrs)
						return
					}
				}
				return
			}
			if len(tt.wantErrs) > 0 {
				t.Fatal("run() succeeded unexpectedly")
			}
		})
	}
}
