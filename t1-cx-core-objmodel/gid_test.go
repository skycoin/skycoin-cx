package objmodel

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
)

func TestGidMasks(t *testing.T) {
	if GidIndexMask|GidHighMask != ^uint64(0) || GidIndexMask&GidHighMask != 0 {
		t.Fatal("index and high masks do not split the Gid")
	}
	if GidGenerationMask|GidTypeTagMask != GidHighMask || GidGenerationMask&GidTypeTagMask != 0 {
		t.Fatal("generation and tag masks do not split the high half")
	}
	if GidIndexBits+GidGenerationBits+GidTypeTagBits != 64 {
		t.Fatal("bit widths do not add up to 64")
	}
	if GidGenerationLast != uint32(GidGenerationMask>>GidGenerationShift) {
		t.Fatal("GidGenerationLast is not the largest generation")
	}
}

func TestGidRoundTrip(t *testing.T) {
	for _, c := range []struct {
		tag   TypeTag
		gen   uint32
		index uint32
	}{
		{GidTypeTagFirst, GidGenerationFirst, 0},
		{0x09, 7, 12345},
		{GidTypeTagLast, GidGenerationLast - 1, ^uint32(0)},
	} {
		gid := GidCompose(c.tag, c.gen, c.index)
		if gid <= GidNil {
			t.Errorf("GidCompose(%#x, %#x, %#x) = %#x, want positive", c.tag, c.gen, c.index, gid)
		}
		if GidTypeTag(gid) != c.tag || GidGeneration(gid) != c.gen || GidIndex(gid) != c.index {
			t.Errorf("round trip of %#x gave tag %#x gen %#x index %#x", gid, GidTypeTag(gid), GidGeneration(gid), GidIndex(gid))
		}
	}
	if GidTypeTag(GidNil) != 0 || GidGeneration(GidNil) != 0 || GidIndex(GidNil) != 0 {
		t.Error("nil Gid has a nonzero part")
	}
	if GidTypeTag(-1) != 0xFF {
		t.Error("a negative Gid must carry a tag above GidTypeTagLast")
	}
}

const gidTestCaseEnv = "CX_OBJMODEL_GID_TEST_CASE"

var gidTestCases = map[string]func(){
	"tag zero":       func() { GidCompose(0, 1, 0) },
	"tag high":       func() { GidCompose(0x80, 1, 0) },
	"generation 0":   func() { GidCompose(1, 0, 0) },
	"generation max": func() { GidCompose(1, GidGenerationLast, 0) },
	"check nil":      func() { GidCheck(GidNil, 1, "T", 4) },
	"check negative": func() { GidCheck(-5, 1, "T", 4) },
	"check tag":      func() { GidCheck(GidCompose(2, 1, 0), 1, "T", 4) },
	"check index":    func() { GidCheck(GidCompose(1, 1, 4), 1, "T", 4) },
}

func TestGidAssertChild(t *testing.T) {
	name := os.Getenv(gidTestCaseEnv)
	if name == "" {
		t.Skip("subprocess only")
	}
	gidTestCases[name]()
	t.Fatal("returned after a failed assert")
}

func TestGidAssert(t *testing.T) {
	for name := range gidTestCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(os.Args[0], "-test.run=^TestGidAssertChild$")
			cmd.Env = append(os.Environ(), gidTestCaseEnv+"="+name)
			var stderr strings.Builder
			cmd.Stderr = &stderr
			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != assert.AssertExitCode {
				t.Fatalf("exit = %v, want status %d\nstderr:\n%s", err, assert.AssertExitCode, stderr.String())
			}
			if !strings.Contains(stderr.String(), "assert failed: precondition") {
				t.Errorf("stderr lacks the precondition assert:\n%s", stderr.String())
			}
		})
	}
}
