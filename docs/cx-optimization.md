# CX optimizations

Each optimization of the CX compiler and runtime has a numbered text file in
[optimizations/](../optimizations). A file gives the place in the code, the
problem, the change, its status (proposed, decided, built) with the decision
number, and the measurement behind it.

| # | optimization | status |
|---|---|---|
| 0001 | [TinyGo wasm build without a scheduler](../optimizations/0001-wasm-scheduler-none.txt) | proposed |
| 0002 | [Return a single result as a plain value](../optimizations/0002-single-result-plain-value.txt) | proposed |
| 0003 | [Pass the frame pointer only to functions that need it](../optimizations/0003-frame-pointer-only-when-needed.txt) | proposed |
| 0004 | [Call depth only for recursive functions](../optimizations/0004-call-depth-only-for-recursion.txt) | decided, D144 |
| 0005 | [Arrays as locals and one LLVM global per CX global](../optimizations/0005-arrays-and-globals-out-of-cx-mem.txt) | proposed |
| 0006 | [Power of two sizes for one instruction bound checks, fast and slow path](../optimizations/0006-power-of-two-sizes.txt) | index check built in #614, D146 |
| 0007 | [Index check as one unsigned compare, the slow path](../optimizations/0007-unsigned-single-compare-bounds.txt) | built in #614 |

The measurements come from hand edits of the IR that `cx build --target=llvm`
writes, rebuilt with clang -O2 and timed with perf stat on an x86-64 desktop,
two code layouts per variant. They show what a change can gain. A file moves
to built when the compiler makes the change and its PR has the numbers.

A new optimization takes the next number: `NNNN-short-name.txt`.
