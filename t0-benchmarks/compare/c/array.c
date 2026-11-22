#include "fnv.h"

#ifndef ARRAY_ROUNDS
#define ARRAY_ROUNDS 400
#endif

int main(void) {
	uint64_t h = FNV_OFFSET;
	int ok = 1;
	int32_t nums[150];
	for (int32_t round = 0; round < ARRAY_ROUNDS; round++) {
		for (int32_t i = 0; i < 150; i++)
			nums[i] = 150 - i;
		int32_t swaps = 0;
		for (int32_t i = 0; i < 149; i++)
			for (int32_t j = 0; j < 149 - i; j++)
				if (nums[j] > nums[j + 1]) {
					int32_t t = nums[j];
					nums[j] = nums[j + 1];
					nums[j + 1] = t;
					swaps++;
				}
		h = fnv(h, (uint64_t)swaps);
		ok = ok && swaps == 11175;
	}
	for (int32_t i = 0; i < 150; i++) {
		h = fnv(h, (uint64_t)nums[i]);
		ok = ok && nums[i] == i + 1;
	}
	return report("array", h, ok);
}
