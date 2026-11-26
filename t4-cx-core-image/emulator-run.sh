#!/bin/sh
# Builds cx-core-image with an image embedded for a TinyGo target, runs it
# in the emulator of that target and exits with the exit code of the run.
#
#   CX_MCU_TARGET=cortex-m-qemu t4-cx-core-image/emulator-run.sh program.cximage
#
# Targets: cortex-m-qemu, riscv-qemu, wasip1, esp32 and esp32c3. Needs
# tinygo (TINYGO) and qemu-system-arm, qemu-system-riscv32, wasmtime, or the
# Espressif QEMU (qemu-system-xtensa, qemu-system-riscv32) on PATH. As the
# image-core runner of the execution records, see docs/Microcontrollers.md:
#   CX_EXECUTION_RECORDS_IMAGE_CORE=$PWD/t4-cx-core-image/emulator-run.sh
set -u
image=${1:?usage: CX_MCU_TARGET=target $0 program.cximage}
target=${CX_MCU_TARGET:?set CX_MCU_TARGET}
tinygo=${TINYGO:-tinygo}
seconds=${CX_MCU_SECONDS:-60}
here=$(cd "$(dirname "$0")" && pwd)
# The package is copied inside the module so its imports resolve. go ./...
# skips a directory whose name starts with an underscore.
work=$(mktemp -d "$(dirname "$here")/_cx-mcu-XXXXXX")
trap 'rm -rf "$work"' EXIT
cp "$here"/*.go "$work"/
rm -f "$work"/*_test.go
cp "$image" "$work/program.cximage"
package=./$(basename "$work")
tags="cxcore cximageembed cximagemarkers"

build() {
	(cd "$(dirname "$here")" && "$tinygo" build -tags="$tags" "$@" "$package") >"$work/build.log" 2>&1 || {
		cat "$work/build.log" >&2
		exit 125
	}
}

case $target in
cortex-m-qemu)
	build -target=cortex-m-qemu -o "$work/fw.elf"
	firmware=$work/fw.elf
	set -- qemu-system-arm -machine lm3s6965evb -semihosting -nographic -kernel "$work/fw.elf"
	;;
riscv-qemu)
	build -target=riscv-qemu -o "$work/fw.elf"
	firmware=$work/fw.elf
	set -- qemu-system-riscv32 -machine virt,aclint=on -smp 4 -nographic -bios none -device virtio-rng-device -kernel "$work/fw.elf"
	;;
wasip1)
	build -target=wasip1 -o "$work/fw.wasm"
	firmware=$work/fw.wasm
	set -- wasmtime run "$work/fw.wasm"
	;;
esp32)
	# A .img output is a full 4 MB flash image.
	[ -n "${CX_MCU_FIRMWARE:-}" ] && build -target=esp32-generic -o "$work/app.bin"
	firmware=$work/app.bin
	build -target=esp32-generic -o "$work/flash.img"
	set -- qemu-system-xtensa -machine esp32 -nographic -drive file="$work/flash.img",if=mtd,format=raw
	;;
esp32c3)
	# The C3 ROM loads the image from flash offset 0, so pad the .bin to
	# 4 MB. QEMU has no USB-Serial-JTAG console, so use UART0.
	build -target=esp32c3-generic -serial=uart -o "$work/app.bin"
	firmware=$work/app.bin
	cp "$work/app.bin" "$work/flash.img"
	truncate -s 4M "$work/flash.img"
	set -- qemu-system-riscv32 -machine esp32c3 -nographic -drive file="$work/flash.img",if=mtd,format=raw
	;;
*)
	echo "unknown CX_MCU_TARGET $target" >&2
	exit 125
	;;
esac

# CX_MCU_FIRMWARE names a file to keep the firmware in, to measure it.
[ -n "${CX_MCU_FIRMWARE:-}" ] && cp "$firmware" "$CX_MCU_FIRMWARE"

# The emulator runs until the exit line, as an ESP32 does not stop QEMU.
timeout "$seconds" "$@" </dev/null >"$work/out" 2>"$work/err" &
emulator=$!
waited=0
while ! grep -q '^cx-core-image-exit ' "$work/out" 2>/dev/null; do
	if ! kill -0 $emulator 2>/dev/null || [ $waited -ge $((seconds * 10)) ]; then
		break
	fi
	sleep 0.1
	waited=$((waited + 1))
done
kill $emulator 2>/dev/null
wait $emulator 2>/dev/null

# Print what the run printed between the begin line and the exit line. The
# runner writes a new line before each, so the last line before the exit
# line is the end of the output without its own new line.
code=$(tr -d '\r' <"$work/out" | awk '
	/^cx-core-image-begin$/ { inside = 1; n = 0; next }
	/^cx-core-image-exit / && inside { for (i = 1; i < n; i++) print line[i]; printf "%s", line[n] > "/dev/stdout"; print $2 > "/dev/stderr"; exit }
	inside { line[++n] = $0 }
' 2>&1 >"$work/stdout")
cat "$work/stdout"
case $code in
'' | *[!0-9]*)
	echo "no exit line from $target:" >&2
	tr -d '\r' <"$work/out" | tail -n 20 >&2
	cat "$work/err" >&2
	exit 125
	;;
esac
exit "$code"
