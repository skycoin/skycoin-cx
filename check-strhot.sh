#!/bin/sh
# Checks the STRHOT rule of docs/StringsHotPath.md. Every string operation
# reachable from the run loop of the old executor has a VIOLATION STRHOT-nnn
# comment above it or above its function, with the code and class that
# strhot-codes.txt lists (#412). With -l it lists every site with its code.
exec go run ./t0-cx-tools-strhot "$@"
