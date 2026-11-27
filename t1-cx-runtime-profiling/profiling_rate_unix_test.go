//go:build unix && !solaris && !tinygo

package profiling

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const profilingRateTestChildPath = "CX_PROFILING_RATE_TEST_CHILD_PATH"

// TestCPUProfileStartOnFileRate runs itself as a child so the test can read
// what the runtime writes to file descriptor 2.
func TestCPUProfileStartOnFileRate(t *testing.T) {
	if path := os.Getenv(profilingRateTestChildPath); path != "" {
		profilingRateTestChildRun(path)
		return
	}

	path := filepath.Join(t.TempDir(), "cpu.pprof")
	child := exec.Command(os.Args[0], "-test.run=^TestCPUProfileStartOnFileRate$")
	child.Env = append(os.Environ(), profilingRateTestChildPath+"="+path)
	var stderr bytes.Buffer
	child.Stderr = &stderr
	if err := child.Run(); err != nil {
		t.Fatalf("child: %v, stderr: %s", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "cannot set cpu profile rate") {
		t.Errorf("runtime warned on stderr: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "stderr restored") {
		t.Errorf("stderr not restored after the profile started: %q", stderr.String())
	}

	period, err := profilingRateTestPeriodRead(path)
	if err != nil {
		t.Fatal(err)
	}
	if period != int64(time.Second/500) {
		t.Errorf("profile period is %dns, want %dns for 500 Hz", period, time.Second/500)
	}
}

func profilingRateTestChildRun(path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	stop, err := CPUProfileStartOnFile(f, 500)
	if err != nil {
		panic(err)
	}
	os.Stderr.WriteString("stderr restored\n")
	if err := stop(); err != nil {
		panic(err)
	}
}

// profilingRateTestPeriodRead returns field 12, period, of the gzipped
// perftools.profiles.Profile message at path.
func profilingRateTestPeriodRead(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	unzipped, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	raw, err := io.ReadAll(unzipped)
	if err != nil {
		return 0, err
	}
	for len(raw) > 0 {
		key, n := binary.Uvarint(raw)
		if n <= 0 {
			return 0, errors.New("bad field key")
		}
		raw = raw[n:]
		field, wire := key>>3, key&7
		switch wire {
		case 0:
			value, n := binary.Uvarint(raw)
			if n <= 0 {
				return 0, errors.New("bad varint")
			}
			if field == 12 {
				return int64(value), nil
			}
			raw = raw[n:]
		case 2:
			length, n := binary.Uvarint(raw)
			if n <= 0 || uint64(len(raw)-n) < length {
				return 0, errors.New("bad length")
			}
			raw = raw[n+int(length):]
		default:
			return 0, errors.New("unexpected wire type")
		}
	}
	return 0, errors.New("no period field")
}
