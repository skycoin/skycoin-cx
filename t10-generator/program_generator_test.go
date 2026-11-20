package generator_test

import (
	"testing"

	cxgenerator "github.com/skycoin/skycoin-cx/t10-generator"
)

func TestProgramGenerator(t *testing.T) {
	tests := []struct {
		scenario    string
		withLiteral bool
	}{
		{
			scenario:    "program without literal",
			withLiteral: false,
		},
		{
			scenario:    "program with literal",
			withLiteral: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
			// Now a constructor call, site S0049.
			cxgenerator.GenerateSampleProgram(t, tc.withLiteral)
		})
	}
}
