package main

// objmodelgenTemplates holds the gid, mng and test templates. Messages are
// built only on failure, so a check costs a compare on the hot path.
const objmodelgenTemplates = `
{{- define "gid" -}}
package ast

import objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"

// {{.Obj}}Gid names one {{.Obj}} in its manager, see docs/ObjectModel.md.
type {{.Obj}}Gid int64

const (
	{{.Obj}}GidTypeTag objmodel.TypeTag = {{.TypeTag}}
	{{.Obj}}GidNil     {{.Obj}}Gid      = 0
)
{{end}}

{{- define "mng" -}}
package ast

import (
	"fmt"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// {{.Obj}}Mng is the only place a {{.Obj}} is allocated and released, a
// linear array of slots, see docs/ObjectModel.md.
type {{.Obj}}Mng struct {
	{{.Obj}}MngObjects     []{{.Obj}} // one per slot, len is the Cap
	{{.Obj}}MngGenerations []uint32 // generation of each slot, 0 while never used
	{{.Obj}}MngLive        []bool // slot holds a live object
	{{.Obj}}MngFree        []uint32 // released slots that can be reused
	{{.Obj}}MngNext        uint32 // first slot never used
	{{.Obj}}MngRetired     uint32 // slots retired at objmodel.GidGenerationLast
	{{.Obj}}MngLen         int // live objects
	{{.Obj}}MngGrowable    bool // grow when full instead of asserting
}

// {{.Obj}}MngInit sets the Cap and whether the manager may grow.
func (manager *{{.Obj}}Mng) {{.Obj}}MngInit(capacity int, growable bool) {
	if manager.{{.Obj}}MngLen != 0 {
		assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}MngInit on a manager holding %d objects", manager.{{.Obj}}MngLen))
	}
	if capacity < 0 || uint64(capacity) > objmodel.GidCapMax {
		assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}MngInit capacity %d outside 0 to %d", capacity, objmodel.GidCapMax))
	}
	*manager = {{.Obj}}Mng{
		{{.Obj}}MngObjects:     make([]{{.Obj}}, capacity),
		{{.Obj}}MngGenerations: make([]uint32, capacity),
		{{.Obj}}MngLive:        make([]bool, capacity),
		{{.Obj}}MngGrowable:    growable,
	}
}

// {{.Obj}}Alloc returns the Gid of a new zero {{.Obj}}. A full manager doubles
// its Cap when it is growable and asserts when it is not.
func (manager *{{.Obj}}Mng) {{.Obj}}Alloc() {{.Obj}}Gid {
	var index uint32
	if free := len(manager.{{.Obj}}MngFree); free > 0 {
		index = manager.{{.Obj}}MngFree[free-1]
		manager.{{.Obj}}MngFree = manager.{{.Obj}}MngFree[:free-1]
	} else {
		capacity := len(manager.{{.Obj}}MngObjects)
		if int(manager.{{.Obj}}MngNext) == capacity {
			if !manager.{{.Obj}}MngGrowable || uint64(capacity) >= objmodel.GidCapMax {
				assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}Alloc on a full manager, Cap %d, growable %t", capacity, manager.{{.Obj}}MngGrowable))
			}
			grown := uint64(2 * capacity)
			if grown == 0 {
				grown = 1
			}
			if grown > objmodel.GidCapMax {
				grown = objmodel.GidCapMax
			}
			objects := make([]{{.Obj}}, grown)
			copy(objects, manager.{{.Obj}}MngObjects)
			manager.{{.Obj}}MngObjects = objects
			generations := make([]uint32, grown)
			copy(generations, manager.{{.Obj}}MngGenerations)
			manager.{{.Obj}}MngGenerations = generations
			live := make([]bool, grown)
			copy(live, manager.{{.Obj}}MngLive)
			manager.{{.Obj}}MngLive = live
		}
		index = manager.{{.Obj}}MngNext
		manager.{{.Obj}}MngNext++
		manager.{{.Obj}}MngGenerations[index] = objmodel.GidGenerationFirst
	}
	manager.{{.Obj}}MngObjects[index] = {{.Obj}}{}
	manager.{{.Obj}}MngLive[index] = true
	manager.{{.Obj}}MngLen++
	if manager.{{.Obj}}MngLen != int(manager.{{.Obj}}MngNext)-len(manager.{{.Obj}}MngFree)-int(manager.{{.Obj}}MngRetired) {
		assert.AssertInvariantTrue(false, fmt.Sprintf("{{.Obj}}Alloc: Len %d is not the live slot count", manager.{{.Obj}}MngLen))
	}
	return {{.Obj}}Gid(objmodel.GidCompose({{.Obj}}GidTypeTag, manager.{{.Obj}}MngGenerations[index], index))
}

// {{.Obj}}Release zeroes the object and frees its slot. The slot's
// generation goes up by one, and a slot reaching objmodel.GidGenerationLast
// is retired instead of reused.
func (manager *{{.Obj}}Mng) {{.Obj}}Release(gid {{.Obj}}Gid) {
	objmodel.GidCheck(int64(gid), {{.Obj}}GidTypeTag, "{{.Obj}}", len(manager.{{.Obj}}MngObjects))
	index := objmodel.GidIndex(int64(gid))
	if !manager.{{.Obj}}MngLive[index] {
		assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}Release of slot %d, which is not live, a double release", index))
	}
	if manager.{{.Obj}}MngGenerations[index] != objmodel.GidGeneration(int64(gid)) {
		assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}Release of a stale Gid %#x, slot %d is at generation %d", int64(gid), index, manager.{{.Obj}}MngGenerations[index]))
	}
	manager.{{.Obj}}MngObjects[index] = {{.Obj}}{}
	manager.{{.Obj}}MngLive[index] = false
	manager.{{.Obj}}MngGenerations[index]++
	if manager.{{.Obj}}MngGenerations[index] == objmodel.GidGenerationLast {
		manager.{{.Obj}}MngRetired++
	} else {
		manager.{{.Obj}}MngFree = append(manager.{{.Obj}}MngFree, index)
	}
	manager.{{.Obj}}MngLen--
	if manager.{{.Obj}}MngLen < 0 || manager.{{.Obj}}MngLen != int(manager.{{.Obj}}MngNext)-len(manager.{{.Obj}}MngFree)-int(manager.{{.Obj}}MngRetired) {
		assert.AssertInvariantTrue(false, fmt.Sprintf("{{.Obj}}Release: Len %d is not the live slot count", manager.{{.Obj}}MngLen))
	}
}

// {{.Obj}}Get returns a borrowed pointer to the object. It is stale after the
// next Alloc on this manager or a Release of gid.
func (manager *{{.Obj}}Mng) {{.Obj}}Get(gid {{.Obj}}Gid) *{{.Obj}} {
	objmodel.GidCheck(int64(gid), {{.Obj}}GidTypeTag, "{{.Obj}}", len(manager.{{.Obj}}MngObjects))
	index := objmodel.GidIndex(int64(gid))
	if !manager.{{.Obj}}MngLive[index] {
		assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}Get of slot %d, which is not live, a stale Gid %#x", index, int64(gid)))
	}
	if manager.{{.Obj}}MngGenerations[index] != objmodel.GidGeneration(int64(gid)) {
		assert.AssertPreconditionTrue(false, fmt.Sprintf("{{.Obj}}Get of a stale Gid %#x, slot %d is at generation %d", int64(gid), index, manager.{{.Obj}}MngGenerations[index]))
	}
	return &manager.{{.Obj}}MngObjects[index]
}

// {{.Obj}}Len is the number of live objects.
func (manager *{{.Obj}}Mng) {{.Obj}}Len() int {
	return manager.{{.Obj}}MngLen
}

// {{.Obj}}Cap is the number of slots, retired ones included.
func (manager *{{.Obj}}Mng) {{.Obj}}Cap() int {
	return len(manager.{{.Obj}}MngObjects)
}

// {{.Obj}}MngTeardown asserts that Len is 0 and drops the arrays, which leaves
// Cap at 0.
func (manager *{{.Obj}}Mng) {{.Obj}}MngTeardown() {
	if manager.{{.Obj}}MngLen != 0 {
		assert.AssertInvariantTrue(false, fmt.Sprintf("{{.Obj}}MngTeardown with %d live objects, a leak", manager.{{.Obj}}MngLen))
	}
	*manager = {{.Obj}}Mng{}
}
{{end}}

{{- define "test" -}}
package ast

import (
	"testing"

	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

func {{.Obj}}MngTestCheck(t *testing.T, manager *{{.Obj}}Mng, step string, wantLen int, wantCap int) {
	t.Helper()
	if manager.{{.Obj}}Len() != wantLen || manager.{{.Obj}}Cap() != wantCap {
		t.Fatalf("%s: Len %d Cap %d, want %d and %d", step, manager.{{.Obj}}Len(), manager.{{.Obj}}Cap(), wantLen, wantCap)
	}
}

func Test{{.Obj}}Mng(t *testing.T) {
	var manager {{.Obj}}Mng
	manager.{{.Obj}}MngInit(2, false)
	{{.Obj}}MngTestCheck(t, &manager, "init", 0, 2)

	first := manager.{{.Obj}}Alloc()
	{{call .Set (printf "manager.%sGet(first)" .Obj) "11"}}
	second := manager.{{.Obj}}Alloc()
	{{call .Set (printf "manager.%sGet(second)" .Obj) "22"}}
	{{.Obj}}MngTestCheck(t, &manager, "two allocs", 2, 2)
	if objmodel.GidTypeTag(int64(first)) != {{.Obj}}GidTypeTag || objmodel.GidGeneration(int64(first)) != objmodel.GidGenerationFirst {
		t.Fatalf("first Gid %#x has the wrong tag or generation", int64(first))
	}
	if {{call .Get (printf "manager.%sGet(first)" .Obj)}} != 11 || {{call .Get (printf "manager.%sGet(second)" .Obj)}} != 22 {
		t.Fatal("Get does not return the allocated objects")
	}

	manager.{{.Obj}}Release(first)
	{{.Obj}}MngTestCheck(t, &manager, "release", 1, 2)
	third := manager.{{.Obj}}Alloc()
	{{.Obj}}MngTestCheck(t, &manager, "reuse", 2, 2)
	if objmodel.GidIndex(int64(third)) != objmodel.GidIndex(int64(first)) {
		t.Fatal("Alloc did not reuse the released slot")
	}
	if objmodel.GidGeneration(int64(third)) != objmodel.GidGeneration(int64(first))+1 {
		t.Fatalf("generation %d after one release, want %d", objmodel.GidGeneration(int64(third)), objmodel.GidGeneration(int64(first))+1)
	}
	if {{call .Get (printf "manager.%sGet(third)" .Obj)}} != 0 {
		t.Fatal("a reused slot is not zeroed")
	}

	manager.{{.Obj}}Release(second)
	manager.{{.Obj}}Release(third)
	{{.Obj}}MngTestCheck(t, &manager, "release all", 0, 2)
	manager.{{.Obj}}MngTeardown()
	{{.Obj}}MngTestCheck(t, &manager, "teardown", 0, 0)
}

func Test{{.Obj}}MngGrowth(t *testing.T) {
	var manager {{.Obj}}Mng
	manager.{{.Obj}}MngInit(0, true)
	gids := make([]{{.Obj}}Gid, 9)
	for i := range gids {
		gids[i] = manager.{{.Obj}}Alloc()
		{{call .Set (printf "manager.%sGet(gids[i])" .Obj) "i + 1"}}
		if manager.{{.Obj}}Len() != i+1 || manager.{{.Obj}}Cap() < manager.{{.Obj}}Len() {
			t.Fatalf("alloc %d: Len %d Cap %d", i, manager.{{.Obj}}Len(), manager.{{.Obj}}Cap())
		}
	}
	{{.Obj}}MngTestCheck(t, &manager, "nine allocs", 9, 16)
	for i, gid := range gids {
		if {{call .Get (printf "manager.%sGet(gid)" .Obj)}} != i+1 {
			t.Fatalf("object %d lost by growth", i)
		}
	}
	for _, gid := range gids {
		manager.{{.Obj}}Release(gid)
	}
	{{.Obj}}MngTestCheck(t, &manager, "release all", 0, 16)
	manager.{{.Obj}}MngTeardown()
	{{.Obj}}MngTestCheck(t, &manager, "teardown", 0, 0)
}

func Test{{.Obj}}MngRetirement(t *testing.T) {
	var manager {{.Obj}}Mng
	manager.{{.Obj}}MngInit(2, false)
	manager.{{.Obj}}Alloc()
	manager.{{.Obj}}MngGenerations[0] = objmodel.GidGenerationLast - 1
	manager.{{.Obj}}Release({{.Obj}}Gid(objmodel.GidCompose({{.Obj}}GidTypeTag, objmodel.GidGenerationLast-1, 0)))
	{{.Obj}}MngTestCheck(t, &manager, "retire", 0, 2)
	if manager.{{.Obj}}MngRetired != 1 || len(manager.{{.Obj}}MngFree) != 0 {
		t.Fatalf("retired %d free %d, want 1 and 0", manager.{{.Obj}}MngRetired, len(manager.{{.Obj}}MngFree))
	}
	next := manager.{{.Obj}}Alloc()
	if objmodel.GidIndex(int64(next)) != 1 {
		t.Fatalf("Alloc after retirement took slot %d, want 1", objmodel.GidIndex(int64(next)))
	}
	manager.{{.Obj}}Release(next)
	manager.{{.Obj}}MngTeardown()
	{{.Obj}}MngTestCheck(t, &manager, "teardown", 0, 0)
}

// {{.Obj}}MngTestAsserts are the cases that must crash, with the assert kind
// and a part of the message.
var {{.Obj}}MngTestAsserts = map[string]struct {
	run  func(manager *{{.Obj}}Mng)
	want string
}{
	"get nil": {func(manager *{{.Obj}}Mng) { manager.{{.Obj}}Get({{.Obj}}GidNil) }, "is nil or negative"},
	"get negative": {func(manager *{{.Obj}}Mng) { manager.{{.Obj}}Get(-1) }, "is nil or negative"},
	"get other type": {func(manager *{{.Obj}}Mng) {
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Get({{.Obj}}Gid(objmodel.GidCompose({{.OtherTag}}, 1, 0)))
	}, "has type tag"},
	"get index at cap": {func(manager *{{.Obj}}Mng) {
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Get({{.Obj}}Gid(objmodel.GidCompose({{.Obj}}GidTypeTag, 1, 1)))
	}, "Cap is 1"},
	"get released": {func(manager *{{.Obj}}Mng) {
		gid := manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Release(gid)
		manager.{{.Obj}}Get(gid)
	}, "which is not live"},
	"get stale": {func(manager *{{.Obj}}Mng) {
		gid := manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Release(gid)
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Get(gid)
	}, "stale Gid"},
	"release double": {func(manager *{{.Obj}}Mng) {
		gid := manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Release(gid)
		manager.{{.Obj}}Release(gid)
	}, "a double release"},
	"release stale": {func(manager *{{.Obj}}Mng) {
		gid := manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Release(gid)
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Release(gid)
	}, "stale Gid"},
	"alloc full": {func(manager *{{.Obj}}Mng) {
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}Alloc()
	}, "on a full manager"},
	"alloc retired full": {func(manager *{{.Obj}}Mng) {
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}MngGenerations[0] = objmodel.GidGenerationLast - 1
		manager.{{.Obj}}Release({{.Obj}}Gid(objmodel.GidCompose({{.Obj}}GidTypeTag, objmodel.GidGenerationLast-1, 0)))
		manager.{{.Obj}}Alloc()
	}, "on a full manager"},
	"init holding objects": {func(manager *{{.Obj}}Mng) {
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}MngInit(1, false)
	}, "MngInit on a manager holding 1 objects"},
	"teardown holding objects": {func(manager *{{.Obj}}Mng) {
		manager.{{.Obj}}Alloc()
		manager.{{.Obj}}MngTeardown()
	}, "invariant violated"},
}

// Test{{.Obj}}MngAssertChild is the subprocess body. It does nothing in a
// normal run.
func Test{{.Obj}}MngAssertChild(t *testing.T) {
	name := objmodelTestCaseName()
	if name == "" {
		t.Skip("subprocess only")
	}
	var manager {{.Obj}}Mng
	manager.{{.Obj}}MngInit(1, false)
	{{.Obj}}MngTestAsserts[name].run(&manager)
	t.Fatal("returned after a failed assert")
}

func Test{{.Obj}}MngAsserts(t *testing.T) {
	for name, testCase := range {{.Obj}}MngTestAsserts {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			objmodelTestAssertSubprocess(t, "Test{{.Obj}}MngAssertChild", name, testCase.want)
		})
	}
}
{{end}}
`
