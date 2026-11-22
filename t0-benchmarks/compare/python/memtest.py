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


MEM_N = 1048576
MEM_WALK = 65536


def main():
    h = FNV_OFFSET
    allbits = 0xFFFFFFFF
    mem = [0] * MEM_N
    n = MEM_N

    bad = 0
    total_sum = 0
    for i in range(n):
        mem[i] = i
    for i in range(n):
        read = mem[i]
        if read != i:
            bad += 1
        total_sum += read
    for i in range(n):
        mem[i] = i ^ allbits
    for i in range(n):
        read = mem[i]
        if read != (i ^ allbits):
            bad += 1
        total_sum += read
    h = fnv(h, bad)
    h = fnv(h, total_sum)
    total = bad

    for bit in range(32):
        pattern = 1 << bit
        bad = 0
        total_sum = 0
        for i in range(MEM_WALK):
            mem[i] = pattern
        for i in range(MEM_WALK):
            read = mem[i]
            if read != pattern:
                bad += 1
            total_sum += read
        h = fnv(h, bad)
        h = fnv(h, total_sum)
        total += bad

    for pattern in (0x00000000, 0xFFFFFFFF, 0x55555555, 0xAAAAAAAA):
        inverse = pattern ^ allbits
        bad = 0
        total_sum = 0
        for i in range(n):
            mem[i] = pattern
        for i in range(n):
            read = mem[i]
            if read != pattern:
                bad += 1
            total_sum += read
            mem[i] = inverse
        for i in range(n - 1, -1, -1):
            read = mem[i]
            if read != inverse:
                bad += 1
            total_sum += read
            mem[i] = pattern
        for i in range(n):
            read = mem[i]
            if read != pattern:
                bad += 1
            total_sum += read
        h = fnv(h, bad)
        h = fnv(h, total_sum)
        total += bad
    return report("memtest", h, total == 0)


sys.exit(main())
