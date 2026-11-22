#include "fnv.h"

#ifndef MEM_N
#define MEM_N 1048576
#define MEM_WALK 65536
#endif

static uint32_t mem[MEM_N];

int main(void) {
	uint64_t h = FNV_OFFSET, sum = 0;
	const uint32_t all = 0xFFFFFFFFu;
	int32_t bad = 0, total;
	uint32_t read;

	for (int32_t i = 0; i < MEM_N; i++)
		mem[i] = (uint32_t)i;
	for (int32_t i = 0; i < MEM_N; i++) {
		read = mem[i];
		if (read != (uint32_t)i)
			bad++;
		sum += read;
	}
	for (int32_t i = 0; i < MEM_N; i++)
		mem[i] = (uint32_t)i ^ all;
	for (int32_t i = 0; i < MEM_N; i++) {
		read = mem[i];
		if (read != ((uint32_t)i ^ all))
			bad++;
		sum += read;
	}
	h = fnv(h, (uint64_t)bad);
	h = fnv(h, sum);
	total = bad;

	for (uint32_t bit = 0; bit < 32; bit++) {
		uint32_t pattern = 1u << bit;
		bad = 0;
		sum = 0;
		for (int32_t i = 0; i < MEM_WALK; i++)
			mem[i] = pattern;
		for (int32_t i = 0; i < MEM_WALK; i++) {
			read = mem[i];
			if (read != pattern)
				bad++;
			sum += read;
		}
		h = fnv(h, (uint64_t)bad);
		h = fnv(h, sum);
		total += bad;
	}

	const uint32_t patterns[4] = {0x00000000u, 0xFFFFFFFFu, 0x55555555u, 0xAAAAAAAAu};
	for (int32_t k = 0; k < 4; k++) {
		uint32_t pattern = patterns[k], inverse = pattern ^ all;
		bad = 0;
		sum = 0;
		for (int32_t i = 0; i < MEM_N; i++)
			mem[i] = pattern;
		for (int32_t i = 0; i < MEM_N; i++) {
			read = mem[i];
			if (read != pattern)
				bad++;
			sum += read;
			mem[i] = inverse;
		}
		for (int32_t i = MEM_N - 1; i >= 0; i--) {
			read = mem[i];
			if (read != inverse)
				bad++;
			sum += read;
			mem[i] = pattern;
		}
		for (int32_t i = 0; i < MEM_N; i++) {
			read = mem[i];
			if (read != pattern)
				bad++;
			sum += read;
		}
		h = fnv(h, (uint64_t)bad);
		h = fnv(h, sum);
		total += bad;
	}
	return report("memtest", h, total == 0);
}
