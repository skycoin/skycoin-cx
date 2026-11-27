package main

// objmodelgenTemplates holds the gid, mng and test templates. Messages are
// built only on failure, so a check costs a compare on the hot path.
const objmodelgenTemplates = `
{{- define "gid" -}}
package ast

import objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"

// {{.Object}}Gid names one {{.Object}} in its manager, see docs/ObjectModel.md.
type {{.Object}}Gid int64

const (
	{{.Object}}GidTypeTag objmodel.TypeTag = {{.TypeTag}}
	{{.Object}}GidNil     {{.Object}}Gid      = 0
)
{{end}}

{{- define "mng" -}}
package ast

import (
	"fmt"

	assert "github.com/skycoin/skycoin-cx/t0-cx-core-assert"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// {{.Object}}Mng is the only place a {{.Object}} is allocated and released, a
// linear array of slots, see docs/ObjectModel.md.
type {{.Object}}Mng struct {
	{{.Object}}MngObjects     []{{.Object}} // one per slot, len is the Cap
	{{.Object}}MngGenerations []uint32 // generation of each slot, 0 while never used
	{{.Object}}MngLive        []bool // slot holds a live object
	{{.Object}}MngFree        []uint32 // released slots that can be reused
	{{.Object}}MngNext        uint32 // first slot never used
	{{.Object}}MngRetired     uint32 // slots retired at objmodel.GidGenerationLast
	{{.Object}}MngLen         int // live objects
	{{.Object}}MngGrowable    bool // the grow boundary may grow it
}

// {{.Object}}MngInit sets the Cap and whether the grow boundary may grow it.
func (manager *{{.Object}}Mng) {{.Object}}MngInit(capacity int, growable bool) {
	if manager.{{.Object}}MngLen != 0 || capacity < 0 || uint64(capacity) > objmodel.GidCapMax {
		objmodel.GidMngInitFailureReport({{.Object}}GidTypeTag, manager.{{.Object}}MngLen, capacity)
	}
	*manager = {{.Object}}Mng{
		{{.Object}}MngObjects:     make([]{{.Object}}, capacity),
		{{.Object}}MngGenerations: make([]uint32, capacity),
		{{.Object}}MngLive:        make([]bool, capacity),
		{{.Object}}MngGrowable:    growable,
	}
}

// {{.Object}}Alloc returns the Gid of a new zero {{.Object}}. It never moves the
// array, so a Get pointer stays valid until the next grow boundary. A full
// manager asserts, an unhandled case for now, see docs/ObjectModel.md.
func (manager *{{.Object}}Mng) {{.Object}}Alloc() {{.Object}}Gid {
	var index uint32
	if free := len(manager.{{.Object}}MngFree); free > 0 {
		index = manager.{{.Object}}MngFree[free-1]
		manager.{{.Object}}MngFree = manager.{{.Object}}MngFree[:free-1]
	} else {
		if capacity := len(manager.{{.Object}}MngObjects); int(manager.{{.Object}}MngNext) == capacity {
			assert.PreconditionTrue(false, fmt.Sprintf("{{.Object}}Alloc on a full manager, Cap %d. Growth between grow boundaries is an unhandled case", capacity))
		}
		index = manager.{{.Object}}MngNext
		manager.{{.Object}}MngNext++
		manager.{{.Object}}MngGenerations[index] = objmodel.GidGenerationFirst
	}
	manager.{{.Object}}MngObjects[index] = {{.Object}}{}
	manager.{{.Object}}MngLive[index] = true
	manager.{{.Object}}MngLen++
	if manager.{{.Object}}MngLen != int(manager.{{.Object}}MngNext)-len(manager.{{.Object}}MngFree)-int(manager.{{.Object}}MngRetired) {
		assert.InvariantTrue(false, fmt.Sprintf("{{.Object}}Alloc: Len %d is not the live slot count", manager.{{.Object}}MngLen))
	}
	return {{.Object}}Gid(objmodel.GidCompose({{.Object}}GidTypeTag, manager.{{.Object}}MngGenerations[index], index))
}

// {{.Object}}Release zeroes the object and frees its slot. The slot's
// generation goes up by one, and a slot reaching objmodel.GidGenerationLast
// is retired instead of reused.
func (manager *{{.Object}}Mng) {{.Object}}Release(gid {{.Object}}Gid) {
	index := objmodel.GidIndex(int64(gid))
	if objmodel.GidTypeTag(int64(gid)) != {{.Object}}GidTypeTag || uint64(index) >= uint64(len(manager.{{.Object}}MngObjects)) ||
		!manager.{{.Object}}MngLive[index] || manager.{{.Object}}MngGenerations[index] != objmodel.GidGeneration(int64(gid)) {
		objmodel.GidAssertFailure(int64(gid), {{.Object}}GidTypeTag, len(manager.{{.Object}}MngObjects))
		if !manager.{{.Object}}MngLive[index] {
			assert.PreconditionTrue(false, fmt.Sprintf("{{.Object}}Release of slot %d, which is not live, a double release", index))
		}
		assert.PreconditionTrue(false, fmt.Sprintf("{{.Object}}Release of a stale Gid %#x, slot %d is at generation %d", int64(gid), index, manager.{{.Object}}MngGenerations[index]))
	}
	manager.{{.Object}}MngObjects[index] = {{.Object}}{}
	manager.{{.Object}}MngLive[index] = false
	manager.{{.Object}}MngGenerations[index]++
	if manager.{{.Object}}MngGenerations[index] == objmodel.GidGenerationLast {
		manager.{{.Object}}MngRetired++
	} else {
		manager.{{.Object}}MngFree = append(manager.{{.Object}}MngFree, index)
	}
	manager.{{.Object}}MngLen--
	if manager.{{.Object}}MngLen < 0 || manager.{{.Object}}MngLen != int(manager.{{.Object}}MngNext)-len(manager.{{.Object}}MngFree)-int(manager.{{.Object}}MngRetired) {
		assert.InvariantTrue(false, fmt.Sprintf("{{.Object}}Release: Len %d is not the live slot count", manager.{{.Object}}MngLen))
	}
}

// {{.Object}}Get returns a borrowed pointer to the object. It is stale after the
// next grow boundary or a Release of gid.
func (manager *{{.Object}}Mng) {{.Object}}Get(gid {{.Object}}Gid) *{{.Object}} {
	index := objmodel.GidIndex(int64(gid))
	if objmodel.GidTypeTag(int64(gid)) != {{.Object}}GidTypeTag || uint64(index) >= uint64(len(manager.{{.Object}}MngObjects)) ||
		!manager.{{.Object}}MngLive[index] || manager.{{.Object}}MngGenerations[index] != objmodel.GidGeneration(int64(gid)) {
		objmodel.GidGetFailureReport(int64(gid), {{.Object}}GidTypeTag, len(manager.{{.Object}}MngObjects), manager.{{.Object}}MngLive, manager.{{.Object}}MngGenerations)
	}
	return &manager.{{.Object}}MngObjects[index]
}

// {{.Object}}Len is the number of live objects.
func (manager *{{.Object}}Mng) {{.Object}}Len() int {
	return manager.{{.Object}}MngLen
}

// {{.Object}}Cap is the number of slots, retired ones included.
func (manager *{{.Object}}Mng) {{.Object}}Cap() int {
	return len(manager.{{.Object}}MngObjects)
}

// {{.Object}}MngGrowBoundary grows a growable manager whose free slots are under
// half of its Cap to the next power of two, and returns the old and new Cap.
// It moves the array, so it runs only where no Get pointer is held.
func (manager *{{.Object}}Mng) {{.Object}}MngGrowBoundary() (int, int) {
	capacity := len(manager.{{.Object}}MngObjects)
	free := capacity - int(manager.{{.Object}}MngNext) + len(manager.{{.Object}}MngFree)
	if !manager.{{.Object}}MngGrowable || 2*free >= capacity && capacity > 0 || uint64(capacity) >= objmodel.GidCapMax {
		return capacity, capacity
	}
	grown := uint64(1)
	for grown <= uint64(capacity) {
		grown *= 2
	}
	if grown > objmodel.GidCapMax {
		grown = objmodel.GidCapMax
	}
	objects := make([]{{.Object}}, grown)
	copy(objects, manager.{{.Object}}MngObjects)
	manager.{{.Object}}MngObjects = objects
	generations := make([]uint32, grown)
	copy(generations, manager.{{.Object}}MngGenerations)
	manager.{{.Object}}MngGenerations = generations
	live := make([]bool, grown)
	copy(live, manager.{{.Object}}MngLive)
	manager.{{.Object}}MngLive = live
	return capacity, int(grown)
}

// {{.Object}}MngTeardown asserts that Len is 0 and drops the arrays, which leaves
// Cap at 0.
func (manager *{{.Object}}Mng) {{.Object}}MngTeardown() {
	if manager.{{.Object}}MngLen != 0 {
		assert.InvariantTrue(false, fmt.Sprintf("{{.Object}}MngTeardown with %d live objects, a leak", manager.{{.Object}}MngLen))
	}
	*manager = {{.Object}}Mng{}
}
{{end}}

{{- define "test" -}}
package ast

import (
	"testing"

	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

func {{.Object}}MngTestCheck(t *testing.T, manager *{{.Object}}Mng, step string, wantLen int, wantCap int) {
	t.Helper()
	if manager.{{.Object}}Len() != wantLen || manager.{{.Object}}Cap() != wantCap {
		t.Fatalf("%s: Len %d Cap %d, want %d and %d", step, manager.{{.Object}}Len(), manager.{{.Object}}Cap(), wantLen, wantCap)
	}
}

func Test{{.Object}}Mng(t *testing.T) {
	var manager {{.Object}}Mng
	manager.{{.Object}}MngInit(2, false)
	{{.Object}}MngTestCheck(t, &manager, "init", 0, 2)

	first := manager.{{.Object}}Alloc()
	{{call .Set (printf "manager.%sGet(first)" .Object) "11"}}
	second := manager.{{.Object}}Alloc()
	{{call .Set (printf "manager.%sGet(second)" .Object) "22"}}
	{{.Object}}MngTestCheck(t, &manager, "two allocs", 2, 2)
	if objmodel.GidTypeTag(int64(first)) != {{.Object}}GidTypeTag || objmodel.GidGeneration(int64(first)) != objmodel.GidGenerationFirst {
		t.Fatalf("first Gid %#x has the wrong tag or generation", int64(first))
	}
	if {{call .Get (printf "manager.%sGet(first)" .Object)}} != 11 || {{call .Get (printf "manager.%sGet(second)" .Object)}} != 22 {
		t.Fatal("Get does not return the allocated objects")
	}

	manager.{{.Object}}Release(first)
	{{.Object}}MngTestCheck(t, &manager, "release", 1, 2)
	third := manager.{{.Object}}Alloc()
	{{.Object}}MngTestCheck(t, &manager, "reuse", 2, 2)
	if objmodel.GidIndex(int64(third)) != objmodel.GidIndex(int64(first)) {
		t.Fatal("Alloc did not reuse the released slot")
	}
	if objmodel.GidGeneration(int64(third)) != objmodel.GidGeneration(int64(first))+1 {
		t.Fatalf("generation %d after one release, want %d", objmodel.GidGeneration(int64(third)), objmodel.GidGeneration(int64(first))+1)
	}
	if {{call .Get (printf "manager.%sGet(third)" .Object)}} != 0 {
		t.Fatal("a reused slot is not zeroed")
	}

	manager.{{.Object}}Release(second)
	manager.{{.Object}}Release(third)
	{{.Object}}MngTestCheck(t, &manager, "release all", 0, 2)
	manager.{{.Object}}MngTeardown()
	{{.Object}}MngTestCheck(t, &manager, "teardown", 0, 0)
}

func Test{{.Object}}MngGrowth(t *testing.T) {
	var manager {{.Object}}Mng
	manager.{{.Object}}MngInit(4, true)
	gids := make([]{{.Object}}Gid, 0, 9)
	for i := 0; i < 2; i++ {
		gids = append(gids, manager.{{.Object}}Alloc())
		{{call .Set (printf "manager.%sGet(gids[i])" .Object) "i + 1"}}
	}
	if oldCap, newCap := manager.{{.Object}}MngGrowBoundary(); oldCap != 4 || newCap != 4 {
		t.Fatalf("half free: grew from %d to %d, want no growth", oldCap, newCap)
	}
	gids = append(gids, manager.{{.Object}}Alloc())
	{{call .Set (printf "manager.%sGet(gids[2])" .Object) "3"}}
	if oldCap, newCap := manager.{{.Object}}MngGrowBoundary(); oldCap != 4 || newCap != 8 {
		t.Fatalf("one free of four: grew from %d to %d, want 4 to 8", oldCap, newCap)
	}
	for i := 3; i < 8; i++ {
		gids = append(gids, manager.{{.Object}}Alloc())
		{{call .Set (printf "manager.%sGet(gids[i])" .Object) "i + 1"}}
	}
	if oldCap, newCap := manager.{{.Object}}MngGrowBoundary(); oldCap != 8 || newCap != 16 {
		t.Fatalf("full: grew from %d to %d, want 8 to 16", oldCap, newCap)
	}
	{{.Object}}MngTestCheck(t, &manager, "eight allocs", 8, 16)
	for i, gid := range gids {
		if {{call .Get (printf "manager.%sGet(gid)" .Object)}} != i+1 {
			t.Fatalf("object %d lost by growth", i)
		}
	}
	for _, gid := range gids {
		manager.{{.Object}}Release(gid)
	}
	{{.Object}}MngTestCheck(t, &manager, "release all", 0, 16)
	manager.{{.Object}}MngTeardown()
	{{.Object}}MngTestCheck(t, &manager, "teardown", 0, 0)

	var fixed {{.Object}}Mng
	fixed.{{.Object}}MngInit(1, false)
	gid := fixed.{{.Object}}Alloc()
	if oldCap, newCap := fixed.{{.Object}}MngGrowBoundary(); oldCap != 1 || newCap != 1 {
		t.Fatalf("fixed manager grew from %d to %d", oldCap, newCap)
	}
	fixed.{{.Object}}Release(gid)
	fixed.{{.Object}}MngTeardown()
}

func Test{{.Object}}MngRetirement(t *testing.T) {
	var manager {{.Object}}Mng
	manager.{{.Object}}MngInit(2, false)
	manager.{{.Object}}Alloc()
	manager.{{.Object}}MngGenerations[0] = objmodel.GidGenerationLast - 1
	manager.{{.Object}}Release({{.Object}}Gid(objmodel.GidCompose({{.Object}}GidTypeTag, objmodel.GidGenerationLast-1, 0)))
	{{.Object}}MngTestCheck(t, &manager, "retire", 0, 2)
	if manager.{{.Object}}MngRetired != 1 || len(manager.{{.Object}}MngFree) != 0 {
		t.Fatalf("retired %d free %d, want 1 and 0", manager.{{.Object}}MngRetired, len(manager.{{.Object}}MngFree))
	}
	next := manager.{{.Object}}Alloc()
	if objmodel.GidIndex(int64(next)) != 1 {
		t.Fatalf("Alloc after retirement took slot %d, want 1", objmodel.GidIndex(int64(next)))
	}
	manager.{{.Object}}Release(next)
	manager.{{.Object}}MngTeardown()
	{{.Object}}MngTestCheck(t, &manager, "teardown", 0, 0)
}

// {{.Object}}MngTestAsserts are the cases that must crash, with the assert kind
// and a part of the message.
var {{.Object}}MngTestAsserts = map[string]struct {
	run  func(manager *{{.Object}}Mng)
	want string
}{
	"get nil": {func(manager *{{.Object}}Mng) { manager.{{.Object}}Get({{.Object}}GidNil) }, "is nil or negative"},
	"get negative": {func(manager *{{.Object}}Mng) { manager.{{.Object}}Get(-1) }, "is nil or negative"},
	"get other type": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}Alloc()
		manager.{{.Object}}Get({{.Object}}Gid(objmodel.GidCompose({{.OtherTag}}, 1, 0)))
	}, "has type tag"},
	"get index at cap": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}Alloc()
		manager.{{.Object}}Get({{.Object}}Gid(objmodel.GidCompose({{.Object}}GidTypeTag, 1, 1)))
	}, "Cap is 1"},
	"get released": {func(manager *{{.Object}}Mng) {
		gid := manager.{{.Object}}Alloc()
		manager.{{.Object}}Release(gid)
		manager.{{.Object}}Get(gid)
	}, "which is not live"},
	"get stale": {func(manager *{{.Object}}Mng) {
		gid := manager.{{.Object}}Alloc()
		manager.{{.Object}}Release(gid)
		manager.{{.Object}}Alloc()
		manager.{{.Object}}Get(gid)
	}, "stale Gid"},
	"release double": {func(manager *{{.Object}}Mng) {
		gid := manager.{{.Object}}Alloc()
		manager.{{.Object}}Release(gid)
		manager.{{.Object}}Release(gid)
	}, "a double release"},
	"release stale": {func(manager *{{.Object}}Mng) {
		gid := manager.{{.Object}}Alloc()
		manager.{{.Object}}Release(gid)
		manager.{{.Object}}Alloc()
		manager.{{.Object}}Release(gid)
	}, "stale Gid"},
	"alloc full": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}Alloc()
		manager.{{.Object}}Alloc()
	}, "on a full manager"},
	"alloc full growable": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}MngInit(1, true)
		manager.{{.Object}}Alloc()
		manager.{{.Object}}Alloc()
	}, "unhandled case"},
	"alloc retired full": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}Alloc()
		manager.{{.Object}}MngGenerations[0] = objmodel.GidGenerationLast - 1
		manager.{{.Object}}Release({{.Object}}Gid(objmodel.GidCompose({{.Object}}GidTypeTag, objmodel.GidGenerationLast-1, 0)))
		manager.{{.Object}}Alloc()
	}, "on a full manager"},
	"init holding objects": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}Alloc()
		manager.{{.Object}}MngInit(1, false)
	}, "MngInit on a manager holding 1 objects"},
	"teardown holding objects": {func(manager *{{.Object}}Mng) {
		manager.{{.Object}}Alloc()
		manager.{{.Object}}MngTeardown()
	}, "invariant violated"},
}

// Test{{.Object}}MngAssertChild is the subprocess body. It does nothing in a
// normal run.
func Test{{.Object}}MngAssertChild(t *testing.T) {
	name := objmodelTestCaseName()
	if name == "" {
		t.Skip("subprocess only")
	}
	var manager {{.Object}}Mng
	manager.{{.Object}}MngInit(1, false)
	{{.Object}}MngTestAsserts[name].run(&manager)
	t.Fatal("returned after a failed assert")
}

func Test{{.Object}}MngAsserts(t *testing.T) {
	for name, testCase := range {{.Object}}MngTestAsserts {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			objmodelTestAssertSubprocess(t, "Test{{.Object}}MngAssertChild", name, testCase.want)
		})
	}
}
{{end}}
`

// objmodelgenTypeTagNameTemplate writes the type name table of package
// objmodel, which names a type tag in an assert message without a string
// parameter on the Get and Release path.
const objmodelgenTypeTagNameTemplate = `package objmodel

// GidTypeTagNameTable names the object type of each type tag. An unused tag
// has an empty name.
var GidTypeTagNameTable = [256]string{
{{- range .}}
	{{printf "0x%02X" .TypeTag}}: "{{.Object}}",
{{- end}}
}
`
