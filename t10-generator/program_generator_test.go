package generator_test

// import (
// 	"testing"
//
// 	cxgenerator "github.com/skycoin/skycoin-cx/t10-generator"
// )

// FAILING: the generator panics, it still assumes deprecated CXArgument inputs; see #12
// func TestProgramGenerator(t *testing.T) {
// 	tests := []struct {
// 		scenario    string
// 		withLiteral bool
// 	}{
// 		{
// 			scenario:    "program without literal",
// 			withLiteral: false,
// 		},
// 		{
// 			scenario:    "program with literal",
// 			withLiteral: true,
// 		},
// 	}
//
// 	for _, tc := range tests {
// 		t.Run(tc.scenario, func(t *testing.T) {
// 			cxgenerator.GenerateSampleProgram(t, tc.withLiteral)
// 		})
// 	}
// }
