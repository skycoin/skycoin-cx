#include "fnv.h"

#ifndef COUNT_N
#define COUNT_N 30000000
#endif

int main(void) {
	int64_t sum = 0;
	for (int32_t i = 0; i < COUNT_N; i++)
		sum += i;
	int64_t want = (int64_t)COUNT_N * (COUNT_N - 1) / 2;
	return report("count", fnv(FNV_OFFSET, (uint64_t)sum), sum == want);
}
