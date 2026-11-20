package actions

import "fmt"

var (
	// VIOLATION: state outside State. Owner: t4-actions; moves to ParseMng in phase 2.
	TemporaryVariableCounter int
)

// generateTempVarName generates tmp name used for temporary variables.
func generateTempVarName(name string) string {
	tempVariableName := fmt.Sprintf("%s_%d", name, TemporaryVariableCounter)
	TemporaryVariableCounter++

	return tempVariableName
}
