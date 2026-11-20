package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	CxPath         string
	WorkingDir     string
	TestsMask      Bits
	LogMask        Bits
	DefaultTimeout time.Duration
}

type TestRunner struct {
	// VIOLATION: pointer at rest; becomes TestConfigGid in phase 3.
	cfg         *Config
	TestCount   int
	TestSuccess int
	TestSkipped int
}

func NewTestRunner(cfg *Config) *TestRunner {
	return &TestRunner{
		cfg: cfg,
	}
}

func (t *TestRunner) Run(args string, exitCode int, desc string) {
	t.RunEx(args, exitCode, desc, TestStable, t.cfg.DefaultTimeout)
}

// RunWant is Run that also requires each of want to appear in the output.
func (t *TestRunner) RunWant(args string, exitCode int, desc string, want ...string) {
	t.runEx(args, exitCode, desc, TestStable, t.cfg.DefaultTimeout, want)
}

func (t *TestRunner) RunEx(args string, exitCode int, desc string, filter Bits, timeout time.Duration) {
	t.runEx(args, exitCode, desc, filter, timeout, nil)
}

func (t *TestRunner) runEx(args string, exitCode int, desc string, filter Bits, timeout time.Duration, want []string) {
	if timeout == 0 {
		timeout = t.cfg.DefaultTimeout
	}

	// A test runs only when every set it belongs to is enabled.
	if !HasAll(t.cfg.TestsMask, filter) {
		if Has(t.cfg.LogMask, LogSkip) {
			fmt.Printf("#--- | SKIPPED | na | '%s' | na | na | %s\\n", args, desc)
		}

		t.TestSkipped = t.TestSkipped + 1
		return
	}

	start := time.Now().Unix()
	out, err := runCmd(t.cfg.CxPath, strings.Split(args, " "), t.cfg.WorkingDir, timeout)
	end := time.Now().Unix()

	timing := "na"
	if Has(t.cfg.LogMask, LogTime) {
		timing = fmt.Sprintf("%dms", end-start)
	}

	t.TestCount += 1

	var ec int
	var stderr []byte
	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Printf("#%s%d | FAILED  | %s | '%s' | exec.Command timeout\n",
				padding(t), t.TestCount, timing, args)
			return
		}

		exitError, ok := err.(*exec.ExitError)
		if !ok {
			fmt.Printf("#%s%d | FAILED  | %s | '%s' | %v\n",
				padding(t), t.TestCount, timing, args, err)
			return
		}
		ec = exitError.ExitCode()
		stderr = exitError.Stderr
	}

	if ec != exitCode {
		if Has(t.cfg.LogMask, LogFail) {
			fmt.Printf("#%s%d | FAILED  | %s | '%s' | exec.Command exited with code %d, expected %d\n",
				padding(t), t.TestCount, timing, args, ec, exitCode)
		}

		if Has(t.cfg.LogMask, LogStderr) {
			fmt.Printf("#%s%d | Stderr: %v, %v\n",
				padding(t), t.TestCount, string(out), string(stderr))
		}
		return
	}

	for _, w := range want {
		if !strings.Contains(string(out), w) && !strings.Contains(string(stderr), w) {
			if Has(t.cfg.LogMask, LogFail) {
				fmt.Printf("#%s%d | FAILED  | %s | '%s' | output does not contain %q\n",
					padding(t), t.TestCount, timing, args, w)
			}
			return
		}
	}

	if Has(t.cfg.LogMask, LogSuccess) {
		fmt.Printf("#%s%d | SUCCESS | %s | '%s' | expected %d | got %d \n",
			padding(t), t.TestCount, timing, args, exitCode, ec)
	}
	t.TestSuccess += 1
}

func runCmd(path string, args []string, dir string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return out, ctx.Err()
	}
	return out, err
}

func padding(t *TestRunner) string {
	var padding string
	if t.TestCount < 10 {
		padding = "  "
	} else if t.TestCount < 100 {
		padding = " "
	}
	return padding
}
