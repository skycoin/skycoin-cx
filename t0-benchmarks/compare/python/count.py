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


COUNT_N = 30000000


def main():
    total = 0
    i = 0
    while i < COUNT_N:
        total += i
        i += 1
    return report("count", fnv(FNV_OFFSET, total), total == COUNT_N * (COUNT_N - 1) // 2)


sys.exit(main())
