package main

import (
	"reflect"
	"testing"
)

func TestEmbeds(t *testing.T) {
	src := `package x

import "io"

type A interface{ Foo() }
type B interface {
	A
	io.Reader
	Bar() io.Writer
}
type C interface{ ~int | string }
type D interface{ int }

func f() { var _ interface{ error } }
`
	got, err := embeds("x.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"x.go B A", "x.go B io.Reader", "x.go anonymous error"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
