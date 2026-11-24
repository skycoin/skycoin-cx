#!/bin/sh
# Checks the STRHOT rule of docs/StringsHotPath.md. Every string operation
# reachable from the run loop of the old executor has a VIOLATION STRHOT-nnn
# comment above it or above its function, with the code and class that
# strhot-codes.txt lists (#412). With -l it lists every site with its code.
#
# Without arguments it checks the default build, then the build with the cxfx
# tag, which adds t6-cxfx (#432), when cgo and the GL, X11 and OpenAL headers
# are there. CX_STRHOT_CXFX=1 makes a missing cxfx build an error, as the
# vet-gui job of CI sets it.
if [ $# -gt 0 ]; then
	exec go run ./t0-cx-tools-strhot "$@"
fi
go run ./t0-cx-tools-strhot || exit 1
if go list -tags cxfx -export ./t6-cxfx >/dev/null 2>&1; then
	go run ./t0-cx-tools-strhot -tags cxfx
elif [ "$CX_STRHOT_CXFX" = 1 ]; then
	echo "check-strhot: the cxfx build does not compile here" >&2
	go list -tags cxfx -export ./t6-cxfx >/dev/null
	exit 1
else
	echo "check-strhot: skipped t6-cxfx, the cxfx build needs cgo and the GL, X11 and OpenAL headers, see make check-strhot-gui"
fi
