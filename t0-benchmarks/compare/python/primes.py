import sys
FNV_OFFSET = 14695981039346656037
FNV_PRIME = 1099511628211
MASK64 = 0xFFFFFFFFFFFFFFFF


def fnv(h, v):
    return ((h ^ (v & MASK64)) * FNV_PRIME) & MASK64


def report(name, h, ok):
    print(h)
    print(name, "pass" if ok else "FAIL")
    return 0 if ok else 1


SIEVE_N = 2000000
SIEVE_WANT = 148933
MR_N = 300000
MR_WANT = 25997
LL_ROUNDS = 20


def powmod(base, exp, n):
    r = 1
    a = base % n
    d = exp
    while d > 0:
        if (d & 1) == 1:
            r = r * a % n
        a = a * a % n
        d >>= 1
    return r


def witness(a, d, s, n):
    composite = False
    if a % n != 0:
        x = powmod(a, d, n)
        if x != 1 and x != n - 1:
            composite = True
            r = 1
            while r < s and composite:
                x = x * x % n
                if x == n - 1:
                    composite = False
                r += 1
    return composite


def miller_rabin(n):
    if n < 2:
        return False
    if n % 2 == 0:
        return n == 2
    d = n - 1
    s = 0
    while d % 2 == 0:
        d //= 2
        s += 1
    return not witness(2, d, s, n) and not witness(7, d, s, n) and not witness(61, d, s, n)


def mulmod(x, y, m):
    r = 0
    a = x % m
    b = y
    while b > 0:
        if (b & 1) == 1:
            r += a
            if r >= m:
                r -= m
        a += a
        if a >= m:
            a -= m
        b >>= 1
    return r


def lucas_lehmer(p):
    m = (1 << p) - 1
    s = 4
    for _ in range(p - 2):
        s = mulmod(s, s, m)
        s = (s + m - 2) % m
    return s == 0


def main():
    h = FNV_OFFSET
    ok = True

    sieve = [0] * SIEVE_N
    for i in range(2, SIEVE_N):
        sieve[i] = 1
    i = 2
    while i * i < SIEVE_N:
        if sieve[i] == 1:
            for j in range(i * i, SIEVE_N, i):
                sieve[j] = 0
        i += 1
    count = 0
    for i in range(2, SIEVE_N):
        count += sieve[i]
    h = fnv(h, count)
    ok = ok and count == SIEVE_WANT

    mr_count = 0
    for k in range(MR_N):
        if miller_rabin(k):
            mr_count += 1
    h = fnv(h, mr_count)
    ok = ok and mr_count == MR_WANT

    exponents = (3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61)
    for _ in range(LL_ROUNDS):
        mask = 0
        for p in exponents:
            if lucas_lehmer(p):
                mask |= 1 << p
                h = fnv(h, 1)
            else:
                h = fnv(h, 0)
        ok = ok and mask == 2305843011361841320
    return report("primes", h, ok)


sys.exit(main())
