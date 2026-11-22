#include "fnv.h"

#ifndef FIB_N
#define FIB_N 34
#define FIB_WANT 5702887
#endif

static int32_t fib(int32_t n) { return n < 2 ? n : fib(n - 1) + fib(n - 2); }

int main(void) {
	int32_t r = fib(FIB_N);
	return report("fib", fnv(FNV_OFFSET, (uint64_t)r), r == FIB_WANT);
}
