#include "fnv.h"

#ifndef STRUCT_ROUNDS
#define STRUCT_ROUNDS 80000
#endif

struct point {
	int32_t x;
	int32_t y;
};

int main(void) {
	uint64_t h = FNV_OFFSET;
	int ok = 1;
	struct point points[100];
	for (int32_t i = 0; i < 100; i++) {
		points[i].x = i;
		points[i].y = 2 * i;
	}
	for (int32_t round = 0; round < STRUCT_ROUNDS; round++) {
		int32_t sum = 0;
		for (int32_t i = 0; i < 100; i++) {
			struct point *p = &points[i];
			sum = sum + p->x + p->y;
		}
		h = fnv(h, (uint64_t)sum);
		ok = ok && sum == 14850;
	}
	return report("struct", h, ok);
}
