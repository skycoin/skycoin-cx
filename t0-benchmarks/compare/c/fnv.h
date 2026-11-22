#include <stdint.h>
#include <stdio.h>

#define FNV_OFFSET 14695981039346656037ULL
#define FNV_PRIME 1099511628211ULL

static uint64_t fnv(uint64_t h, uint64_t v) { return (h ^ v) * FNV_PRIME; }

static int report(const char *name, uint64_t h, int ok) {
	printf("%llu\n%s %s\n", (unsigned long long)h, name, ok ? "pass" : "FAIL");
	return ok ? 0 : 1;
}
