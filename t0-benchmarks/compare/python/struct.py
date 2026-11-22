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


STRUCT_ROUNDS = 80000


class Point:
    __slots__ = ("x", "y")

    def __init__(self, x, y):
        self.x = x
        self.y = y


def main():
    h = FNV_OFFSET
    ok = True
    points = [Point(i, 2 * i) for i in range(100)]
    for _ in range(STRUCT_ROUNDS):
        total = 0
        for i in range(100):
            p = points[i]
            total = total + p.x + p.y
        h = fnv(h, total)
        ok = ok and total == 14850
    return report("struct", h, ok)


sys.exit(main())
