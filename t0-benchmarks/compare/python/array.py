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


ARRAY_ROUNDS = 400


def main():
    h = FNV_OFFSET
    ok = True
    nums = [0] * 150
    for _ in range(ARRAY_ROUNDS):
        for i in range(150):
            nums[i] = 150 - i
        swaps = 0
        for i in range(149):
            for j in range(149 - i):
                if nums[j] > nums[j + 1]:
                    t = nums[j]
                    nums[j] = nums[j + 1]
                    nums[j + 1] = t
                    swaps += 1
        h = fnv(h, swaps)
        ok = ok and swaps == 11175
    for i in range(150):
        h = fnv(h, nums[i])
        ok = ok and nums[i] == i + 1
    return report("array", h, ok)


sys.exit(main())
