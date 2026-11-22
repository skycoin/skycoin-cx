package runner

const (
	TestNone   Bits = 0
	TestStable Bits = 1 << iota
	TestIssue
	TestGui
	TestExamples
	// TestPtr32 is a test that fails on a ptr64 build for a known bug.
	TestPtr32
	TestAll = TestStable | TestIssue | TestGui | TestExamples | TestPtr32
)

var TestBits = map[string]Bits{
	"all":      TestAll,
	"stable":   TestStable,
	"issue":    TestIssue,
	"gui":      TestGui,
	"examples": TestExamples,
	"ptr32":    TestPtr32,
}

const (
	LogNone    Bits = 0
	LogSuccess Bits = 1 << iota
	LogStderr
	LogFail
	LogSkip
	LogTime
	LogAll = LogSuccess | LogStderr | LogFail | LogSkip | LogTime
)

var LogBits = map[string]Bits{
	"all":     LogAll,
	"success": LogSuccess,
	"stderr":  LogStderr,
	"fail":    LogFail,
	"skip":    LogSkip,
	"time":    LogTime,
}
