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


FIB_N = 34
FIB_WANT = 5702887


def fib(n):
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)


def main():
    r = fib(FIB_N)
    return report("fib", fnv(FNV_OFFSET, r), r == FIB_WANT)


sys.exit(main())
