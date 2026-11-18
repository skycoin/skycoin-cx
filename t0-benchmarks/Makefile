CX ?= ../bin/cx
CXBENCH := ackermann-recurssive digital-root factorial-iterative factorial-recursive

all: cx go

cx:
	@for b in $(CXBENCH); do echo "== $$b"; time $(CX) cx/$$b.cx; done

go:
	go test -run xxx -bench . ./.

.PHONY: all cx go
