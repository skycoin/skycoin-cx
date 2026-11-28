package gidtag

import "testing"

// TestTableTags checks every tag of Table: no two share a value or a name,
// each lies in its range, no range is reserved, and NameTable agrees.
func TestTableTags(t *testing.T) {
	values := map[uint8]string{}
	names := map[string]bool{}
	for _, tag := range Table {
		if other, taken := values[tag.TagValue]; taken {
			t.Errorf("tag %#x of %s is taken by %s", tag.TagValue, tag.TagName, other)
		}
		values[tag.TagValue] = tag.TagName
		if names[tag.TagName] {
			t.Errorf("tag name %s is used twice", tag.TagName)
		}
		names[tag.TagName] = true
		if tag.TagValue < tag.TagRange.RangeFirst || tag.TagValue > tag.TagRange.RangeLast {
			t.Errorf("tag %#x of %s is outside its range %s, %#x to %#x",
				tag.TagValue, tag.TagName, tag.TagRange.RangeName, tag.TagRange.RangeFirst, tag.TagRange.RangeLast)
		}
		if tag.TagRange == RangeReserved {
			t.Errorf("tag %#x of %s is in the reserved range", tag.TagValue, tag.TagName)
		}
		if tag.TagVariant < VariantObject || tag.TagVariant > VariantLibraryResource || tag.TagOwner == "" {
			t.Errorf("tag %#x of %s has variant %d and owner %q", tag.TagValue, tag.TagName, tag.TagVariant, tag.TagOwner)
		}
		if NameTable[tag.TagValue] != tag.TagName {
			t.Errorf("NameTable names tag %#x %q, want %s, run make generate-gidtags", tag.TagValue, NameTable[tag.TagValue], tag.TagName)
		}
	}
	for value, name := range NameTable {
		if name != "" && values[uint8(value)] != name {
			t.Errorf("NameTable names tag %#x %s, which is not in Table", value, name)
		}
	}
}

// TestRangeTable checks that the ranges run from 0x01 to 0x7F in order
// without a gap or an overlap.
func TestRangeTable(t *testing.T) {
	next := 0x01
	for _, tagRange := range RangeTable {
		if int(tagRange.RangeFirst) != next || tagRange.RangeLast < tagRange.RangeFirst {
			t.Errorf("range %s is %#x to %#x, want it to start at %#x", tagRange.RangeName, tagRange.RangeFirst, tagRange.RangeLast, next)
		}
		next = int(tagRange.RangeLast) + 1
	}
	if next != 0x80 {
		t.Errorf("the ranges end at %#x, want 0x7F", next-1)
	}
}
