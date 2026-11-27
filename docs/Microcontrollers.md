# TinyGo and microcontrollers

This page describes the TinyGo builds of cx: the vm with a program image on
a microcontroller, native programs of the LLVM target linked by TinyGo, and
cx itself. They are an alternative to the Go builds, which stay the default
everywhere.

## The vm on a microcontroller

`cx build --target=image` writes the `ir.Image` that vm runs, the image of
[IR.md](IR.md#the-runtime-image), to a file. `t4-cx-command-image` is a vm
runner with no parser, compiler or AST, which runs such a file. It imports
the vm, the image and the constants only, so it builds with TinyGo for a
microcontroller.

```
./bin/cx build --target=image -stack-size=16K program.cx -o program.cximage
go build -tags=cxcore -o cx-core-image ./t4-cx-command-image
./cx-core-image program.cximage
```

On a board the image is embedded. The `cximageembed` tag embeds
`t4-cx-command-image/program.cximage` as a string, which stays in flash:

```
make build-mcu MCU_TARGET=pico MCU_PROGRAM=tests/examples/hello-world.cx
tinygo flash -target=pico -tags="cxcore cximageembed" ./t4-cx-command-image
```

* The image takes the programs vm takes, and refuses one that uses the file
  or net library, which the runner does not link. The cipher library runs
  inside vm and works.
* The default stack of an image is 1 MB, as for cx. `-stack-size` sets it at
  build time, since the data segment starts after the stack. A
  microcontroller needs a few KB.
* A runtime error prints the error line a native program prints, with no
  call stack, as the image holds no variable values.
* Image opcodes are renumbered between cx versions. The file stores its
  format version and the opcode count, and the runner refuses a file another
  cx version wrote.
* The runner trusts the image as it trusts a built program. It does not
  check the operands of every instruction.

### In emulators

`make run-mcu MCU_TARGET=<target>` builds the image of MCU_PROGRAM, embeds
it, and runs it with `t4-cx-command-image/emulator-run.sh` in the emulator of
the target. The script reads the output between two marker lines that the
`cximagemarkers` tag prints, because the ESP32 ROM prints boot text and
because QEMU passes no exit code from a Cortex-M or an ESP32.

| MCU_TARGET | emulator |
| --- | --- |
| cortex-m-qemu | qemu-system-arm, lm3s6965evb, Cortex-M3 |
| riscv-qemu | qemu-system-riscv32, virt |
| wasip1 | wasmtime |
| esp32c3 | Espressif QEMU, qemu-system-riscv32 -machine esp32c3 |
| esp32 | Espressif QEMU, qemu-system-xtensa; needs a TinyGo whose LLVM has the Xtensa backend |

The script is also a runner for the `image-core` form of the execution
records, see [ExecutionRecords.md](ExecutionRecords.md):

```
CX_EXECUTION_RECORDS_IMAGE_CORE=$PWD/t4-cx-command-image/emulator-run.sh \
CX_EXECUTION_RECORDS_IMAGE_FLAGS=-stack-size=16K CX_MCU_TARGET=cortex-m-qemu \
go test -count=1 -run '^TestExecutionRecords$' ./t11-cx-command-cx
```

With TinyGo 0.45.0-dev of the 0magnet fork and `-stack-size=16K`, all 236
records of the image-core form pass on each of the five targets. wasip1
needs `wasm-opt` (`WASMOPT`). The Cortex-M3 build is 80 KB of flash and
6 KB of static RAM.

### On a board

`make flash-mcu MCU_TARGET=<target> MCU_PORT=<port>` builds the image of
MCU_PROGRAM into the runner with the marker lines and flashes it with
`tinygo flash`. TinyGo finds the port when MCU_PORT is empty.

```
make flash-mcu MCU_TARGET=xiao-esp32c6 MCU_PORT=/dev/ttyACM0
```

On an ESP32-C6 board, flashed and read over the USB-Serial-JTAG port of the
chip with TinyGo 0.44.0-dev of the 0magnet fork and `-stack-size=16K`, all
236 records of the image-core form pass. The runner is 104 to 123 KB of
flash and 6 KB of static RAM, and each record but one finishes within a
few milliseconds.

* The program runs once at reset, and the USB-Serial-JTAG port drops what
  the chip writes while no host reads it. Open the port first and then reset
  the chip, by its button or by pulsing RTS with DTR low.
* The ROM prints `SHA-256 comparison failed` for a TinyGo image on the C6
  and boots it anyway.
* `-serial=uart` runs UART0 at half the baud rate with a TinyGo before
  commit 64056730, which selects the 80 MHz UART clock on the C6.

## The LLVM target with TinyGo

`cx build --target=llvm` compiles with the clang on the PATH, see
[LlvmTarget.md](LlvmTarget.md). `--llvm-linker=tinygo` uses TinyGo instead,
and cx picks it by itself when there is no clang but a `tinygo` (`CX_TINYGO`
names another). It compiles the module with `tinygo clang` and builds it,
with the C runtime, into a small TinyGo program whose main calls the
program. TinyGo then brings the C library, compiler-rt, startup code and
linker script of the target.

`--llvm-tinygo-target=<target>` cross compiles for a TinyGo target, such as
`pico` or `cortex-m-qemu`. `CX_TINYGO_FLAGS` adds flags to `tinygo build`, such
as `-serial=uart`. The triple, cpu and features come from
`tinygo info -json`, and the extension of `-o` picks the file format, as
for `tinygo build`: `.elf`, `.hex`, `.bin` or `.uf2`.

```
./bin/cx build --target=llvm --llvm-tinygo-target=pico -stack-size=4K hello.cx -o hello.uf2
```

The TinyGo program leaves out the net library of the C runtime, so a
program that uses it is refused. The runtime defines `CX_RT_TINYGO`, which
shrinks its output buffer to 1 KB and ends the process through `os.Exit`.

TinyGo builds picolibc with a printf for integers only, so the runtime
writes integers itself and, in a TinyGo program, formats floats with
`strconv.AppendFloat` on the Go side, as vm does.

A hello world, a division by zero and the fib benchmark, cross compiled
this way, print what vm prints on cortex-m-qemu, riscv-qemu, wasip1 and,
in Espressif QEMU, esp32c3-generic and esp32-generic. A pico build links
to a 25 KB UF2, which has run on no board yet. QEMU passes no exit code
from a Cortex-M or an ESP32.

## cx with TinyGo

`make build-tinygo` builds cx and cx-core with TinyGo, and
`make build-wasm-tinygo` builds cx for js/wasm. `TINYGO` names the tinygo
to run.

TinyGo has no `os.Root` and no `os.SameFile`, has no CPU profiler and cannot
link the amd64 assembly of `lukechampine.com/blake3`. A TinyGo build
therefore uses, by the `tinygo` build tag:

* a `sandbox.Root` that checks each path in Go before the call: it must be
  local and no part of it may be a symlink, except the last one for a
  remove. A symlink made between the check and the call is not caught,
  which `os.Root` catches;
* a portable BLAKE3 in `t0-cx-core-digest`, which a Go test checks against
  `lukechampine.com/blake3`;
* a path comparison with symlinks resolved in place of `os.SameFile`;
* no CPU profile rate.

The Go builds keep `os.Root`, `os.SameFile` and `lukechampine.com/blake3`.

With TinyGo 0.45.0-dev of the 0magnet fork on x86-64 Linux, a TinyGo cx
passes every execution record in every form, run with
`CX_EXECUTION_RECORDS_CX` (see [ExecutionRecords.md](ExecutionRecords.md)).

The js/wasm build compiles but does not run in the playground: the js/wasm
runtime of upstream TinyGo passes no arguments, environment or file system
from JavaScript.
