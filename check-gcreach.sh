#!/bin/sh
# Checks the GC reach rules of docs/GarbageCollector.md. The vm and the IR
# base import neither t3-ast nor the collector, only t3-ast imports the
# collector, and every other reach into the heap and the collector has a
# GCREACH comment directly above it or above its function. With -l it also
# lists every reach as path:line, for the table in the document.
export LC_ALL=C
rc=0
list=0
[ "$1" = -l ] && list=1
for dir in t2-cx-runtime-ir t3-cx-runtime-vm; do
	for file in "$dir"/*.go; do
		case "$file" in *_test.go) continue ;; esac
		if grep -n -e '"github.com/skycoin/skycoin-cx/t3-ast"' -e '"github.com/skycoin/skycoin-cx/t2-cx-runtime-gc"' "$file"; then
			echo "$file imports t3-ast or the collector. $dir stays GC free."
			rc=1
		fi
	done
done
importers=$(find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -name '*.go' -print |
	sed 's#^\./##' | grep -v -e '_test\.go$' -e '^t3-ast/' -e '^t2-cx-runtime-gc/' |
	xargs grep -l '"github.com/skycoin/skycoin-cx/t2-cx-runtime-gc"' /dev/null)
for file in $importers; do
	echo "$file imports t2-cx-runtime-gc. Only t3-ast hands the collector its roots."
	rc=1
done
find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -name '*.go' -print |
	sed 's#^\./##' | grep -v -e '_test\.go$' -e '^t2-cx-runtime-gc/' |
	grep -v -x 't3-ast/gc_type.go' | sort |
	xargs awk -v list="$list" '
function reach(line) {
	if (line ~ /"github\.com\/skycoin\/skycoin-cx\/t3-ast"/)
		return FILENAME ~ /^(t4-execute|t6-cx-runtime-core|t4-cx-compiler-lower|t4-opcodes)\//
	if (line ~ /"github\.com\/skycoin\/skycoin-cx\/t2-cx-runtime-gc"/)
		return 0
	if (line ~ /^func (\([^)]*\) )?(GarbageCollectorCheck|GarbageCollectorTrigger|AllocateSeq|AllocateStr|AllocateStrAt|EscapedAllocate|SliceResizeEx|ResizeMemory|gcFrameRoots|gcGlobalRoots)\(/)
		return 0
	if (line ~ /(^|[^A-Za-z0-9_])(GarbageCollectorCheck|GarbageCollectorTrigger|GarbageCollectorCollect|AllocateSeq|AllocateStr|AllocateStrAt|EscapedAllocate|SliceResizeEx|ResizeMemory|gcFrameRoots|gcGlobalRoots)\(/)
		return 1
	if (line ~ /(^|[^A-Za-z0-9_])(ListOfSlots|ListOfPointers)([^A-Za-z0-9_]|$)/)
		return 1
	return line ~ /Heap\.(Pressure|PressureLimit|Types|Roots|Collector)([^A-Za-z0-9_]|$)/
}
FNR == 1 { block = 0; funccovered = 0; infunc = 0 }
/^[ \t]*\/\// { if ($0 ~ /GCREACH:/) block = 1; next }
/^[ \t]*$/ { block = 0; next }
{
	if (reach($0)) {
		if (block || (infunc && funccovered)) {
			covered++
			if (list) print FILENAME ":" FNR
		}
		else {
			print FILENAME ":" FNR ": GC reach without a GCREACH comment: " $0
			missing++
		}
	}
	if ($0 ~ /^func /) { infunc = 1; funccovered = block }
	if ($0 ~ /^}/) infunc = 0
	block = 0
}
END {
	print covered + 0 " GC reaches with a GCREACH comment, see docs/GarbageCollector.md"
	exit missing > 0
}' || rc=1
exit $rc
