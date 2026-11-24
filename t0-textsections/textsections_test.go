package textsections

import (
	"reflect"
	"testing"
)

func TestTextSectionsRoundTrip(t *testing.T) {
	sections := []TextSection{
		{TextSectionName: "format", TextSectionColumns: []string{"version"}, TextSectionEntries: []TextSectionEntry{{3, "1"}}},
		{TextSectionName: "empty", TextSectionColumns: []string{"key"}},
		{TextSectionName: "names", TextSectionColumns: []string{"key", "name"}, TextSectionEntries: []TextSectionEntry{{8, `a "one"`}, {9, `b "two words"`}}},
	}
	content := TextSectionsWrite([]string{"a comment"}, sections)
	want := "# a comment\n[format] version\n1\n\n[empty] key\n\n[names] key name\na \"one\"\nb \"two words\"\n"
	if string(content) != want {
		t.Fatalf("written %q, want %q", content, want)
	}
	got, err := TextSectionsRead(content)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sections) {
		t.Errorf("read %+v, want %+v", got, sections)
	}
	if names, found := TextSectionFind(got, "names"); !found || len(names.TextSectionEntries) != 2 {
		t.Errorf("TextSectionFind names = %+v, %v", names, found)
	}
}

func TestTextSectionsReadErrors(t *testing.T) {
	for _, content := range []string{
		"entry\n",
		"[a] x\n1\n[b] y\n",
		"[a x\n",
	} {
		if _, err := TextSectionsRead([]byte(content)); err == nil {
			t.Errorf("%q: want an error", content)
		}
	}
}
