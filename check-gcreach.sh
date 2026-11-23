#!/bin/sh
# Checks the GC reach rules of docs/GarbageCollector.md. The vm and the IR
# base never import t3-ast. Every other reach into the heap and the collector
# has a GCREACH comment directly above it or above its function. With -l it
# also lists every reach as path:line, for the table in the document.
export LC_ALL=C
rc=0
list=0
[ "$1" = -l ] && list=1
for dir in t2-cx-runtime-ir t3-cx-runtime-vm; do
	for file in "$dir"/*.go; do
		case "$file" in *_test.go) continue ;; esac
		if grep -n '"github.com/skycoin/skycoin-cx/t3-ast"' "$file"; then
			echo "$file imports t3-ast, which holds the GC. $dir stays GC free."
			rc=1
		fi
	done
done
find . -path ./vendor -prune -o -path ./.git -prune -o -path ./.claude -prune -o -name '*.go' -print |
	sed 's#^\./##' | grep -v '_test\.go$' |
	grep -v -x -e 't3-ast/garbage_collector.go' -e 't3-ast/gc_type.go' | sort |
	xargs awk -v list="$list" '
function reach(line) {
	if (line ~ /"github\.com\/skycoin\/skycoin-cx\/t3-ast"/)
		return FILENAME ~ /^(t4-execute|t6-cx-runtime-core|t4-cx-compiler-lower|t4-opcodes)\//
	if (line ~ /^func (\([^)]*\) )?(MarkAndCompact|AllocateSeq|AllocateStr|AllocateStrAt|EscapedAllocate|SliceResizeEx|ResizeMemory|gcFrameRoots|gcGlobalRoots|AllocWrite_obj_data|AllocWrite_str_data)\(/)
		return 0
	if (line ~ /(^|[^A-Za-z0-9_])(MarkAndCompact|AllocateSeq|AllocateStr|AllocateStrAt|EscapedAllocate|SliceResizeEx|ResizeMemory|gcFrameRoots|gcGlobalRoots|AllocWrite_obj_data|AllocWrite_str_data)\(/)
		return 1
	if (line ~ /(^|[^A-Za-z0-9_])Allocator([^A-Za-z0-9_]|$)/)
		return 1
	if (line ~ /(^|[^A-Za-z0-9_])(ListOfSlots|ListOfPointers)([^A-Za-z0-9_]|$)/)
		return 1
	return line ~ /Heap\.(Pinned|Types|Visited|Slots|Work|ObjectStarts)([^A-Za-z0-9_]|$)/
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
