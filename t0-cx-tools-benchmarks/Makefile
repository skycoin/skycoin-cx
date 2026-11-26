CX ?= ../bin/cx
CXBENCH := ackermann digitalroot factorialiter factorialrec structs structserialize

all: cx go

cx:
	@for b in $(CXBENCH); do echo "== $$b"; time $(CX) compare/cx/$$b.cx; done

go:
	go test -run xxx -bench . ./.

.PHONY: all cx go
