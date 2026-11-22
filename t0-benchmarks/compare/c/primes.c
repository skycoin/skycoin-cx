#include "fnv.h"

#ifndef SIEVE_N
#define SIEVE_N 2000000
#define SIEVE_WANT 148933
#define MR_N 300000
#define MR_WANT 25997
#define LL_ROUNDS 20
#endif

static int32_t sieve[SIEVE_N];

static uint64_t powmod(uint64_t base, uint64_t exp, uint64_t n) {
	uint64_t r = 1, a = base % n, d = exp;
	while (d > 0) {
		if ((d & 1) == 1)
			r = r * a % n;
		a = a * a % n;
		d >>= 1;
	}
	return r;
}

static int witness(uint64_t a, uint64_t d, uint64_t s, uint64_t n) {
	int composite = 0;
	if (a % n != 0) {
		uint64_t x = powmod(a, d, n);
		if (x != 1 && x != n - 1) {
			composite = 1;
			for (uint64_t r = 1; r < s && composite; r++) {
				x = x * x % n;
				if (x == n - 1)
					composite = 0;
			}
		}
	}
	return composite;
}

static int miller_rabin(uint64_t n) {
	if (n < 2)
		return 0;
	if (n % 2 == 0)
		return n == 2;
	uint64_t d = n - 1, s = 0;
	while (d % 2 == 0) {
		d /= 2;
		s++;
	}
	return !witness(2, d, s, n) && !witness(7, d, s, n) && !witness(61, d, s, n);
}

static uint64_t mulmod(uint64_t x, uint64_t y, uint64_t m) {
	uint64_t r = 0, a = x % m, b = y;
	while (b > 0) {
		if ((b & 1) == 1) {
			r += a;
			if (r >= m)
				r -= m;
		}
		a += a;
		if (a >= m)
			a -= m;
		b >>= 1;
	}
	return r;
}

static int lucas_lehmer(uint64_t p) {
	uint64_t m = (1ULL << p) - 1, s = 4;
	for (uint64_t i = 0; i < p - 2; i++) {
		s = mulmod(s, s, m);
		s = (s + m - 2) % m;
	}
	return s == 0;
}

int main(void) {
	uint64_t h = FNV_OFFSET;
	int ok = 1;

	for (int32_t i = 2; i < SIEVE_N; i++)
		sieve[i] = 1;
	for (int32_t i = 2; i * i < SIEVE_N; i++)
		if (sieve[i] == 1)
			for (int32_t j = i * i; j < SIEVE_N; j += i)
				sieve[j] = 0;
	int32_t count = 0;
	for (int32_t i = 2; i < SIEVE_N; i++)
		count += sieve[i];
	h = fnv(h, (uint64_t)count);
	ok = ok && count == SIEVE_WANT;

	int32_t mr_count = 0;
	for (uint64_t k = 0; k < MR_N; k++)
		if (miller_rabin(k))
			mr_count++;
	h = fnv(h, (uint64_t)mr_count);
	ok = ok && mr_count == MR_WANT;

	const uint64_t exponents[17] = {3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61};
	for (int32_t round = 0; round < LL_ROUNDS; round++) {
		uint64_t mask = 0;
		for (int32_t i = 0; i < 17; i++) {
			if (lucas_lehmer(exponents[i])) {
				mask |= 1ULL << exponents[i];
				h = fnv(h, 1);
			} else {
				h = fnv(h, 0);
			}
		}
		ok = ok && mask == 2305843011361841320ULL;
	}
	return report("primes", h, ok);
}
