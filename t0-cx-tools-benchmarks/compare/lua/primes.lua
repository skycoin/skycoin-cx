-- A sieve below 2,000,000, Miller-Rabin below 300,000 and Lucas-Lehmer to 61.
-- Needs Lua 5.4 or later for 64 bit integers and bit operators.

local FNV_OFFSET = 0xcbf29ce484222325 -- 14695981039346656037
local FNV_PRIME = 1099511628211

-- fnv folds one 64 bit word. Lua integers wrap like a ui64.
local function fnv(h, v)
	return (h ~ v) * FNV_PRIME
end

-- report prints the hash as an unsigned 64 bit number and the pass line.
local function report(name, h, ok)
	if h >= 0 then
		print(string.format("%d", h))
	else
		local q = (h >> 1) // 5
		print(string.format("%d%d", q, h - q * 10))
	end
	print(name .. (ok and " pass" or " FAIL"))
	return ok and 0 or 1
end

local SIEVE_N = 2000000
local SIEVE_WANT = 148933
local MR_N = 300000
local MR_WANT = 25997
local LL_ROUNDS = 20

local function powmod(base, exp, n)
	local r = 1
	local a = base % n
	local d = exp
	while d > 0 do
		if (d & 1) == 1 then
			r = r * a % n
		end
		a = a * a % n
		d = d >> 1
	end
	return r
end

local function witness(a, d, s, n)
	local composite = false
	if a % n ~= 0 then
		local x = powmod(a, d, n)
		if x ~= 1 and x ~= n - 1 then
			composite = true
			local r = 1
			while r < s and composite do
				x = x * x % n
				if x == n - 1 then
					composite = false
				end
				r = r + 1
			end
		end
	end
	return composite
end

local function millerRabin(n)
	if n < 2 then
		return false
	end
	if n % 2 == 0 then
		return n == 2
	end
	local d = n - 1
	local s = 0
	while d % 2 == 0 do
		d = d // 2
		s = s + 1
	end
	return not witness(2, d, s, n) and not witness(7, d, s, n) and not witness(61, d, s, n)
end

-- mulmod doubles, so every value stays below 2^62.
local function mulmod(x, y, m)
	local r = 0
	local a = x % m
	local b = y
	while b > 0 do
		if (b & 1) == 1 then
			r = r + a
			if r >= m then
				r = r - m
			end
		end
		a = a + a
		if a >= m then
			a = a - m
		end
		b = b >> 1
	end
	return r
end

local function lucasLehmer(p)
	local m = (1 << p) - 1
	local s = 4
	for _ = 1, p - 2 do
		s = mulmod(s, s, m)
		s = (s + m - 2) % m
	end
	return s == 0
end

local function main()
	local h = FNV_OFFSET
	local ok = true

	local sieve = {}
	sieve[1] = 0
	for i = 2, SIEVE_N - 1 do
		sieve[i] = 1
	end
	local i = 2
	while i * i < SIEVE_N do
		if sieve[i] == 1 then
			for j = i * i, SIEVE_N - 1, i do
				sieve[j] = 0
			end
		end
		i = i + 1
	end
	local count = 0
	for k = 2, SIEVE_N - 1 do
		count = count + sieve[k]
	end
	h = fnv(h, count)
	ok = ok and count == SIEVE_WANT

	local mrCount = 0
	for k = 0, MR_N - 1 do
		if millerRabin(k) then
			mrCount = mrCount + 1
		end
	end
	h = fnv(h, mrCount)
	ok = ok and mrCount == MR_WANT

	local exponents = { 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61 }
	for _ = 1, LL_ROUNDS do
		local mask = 0
		for _, p in ipairs(exponents) do
			if lucasLehmer(p) then
				mask = mask | (1 << p)
				h = fnv(h, 1)
			else
				h = fnv(h, 0)
			end
		end
		ok = ok and mask == 2305843011361841320
	end
	return report("primes", h, ok)
end

os.exit(main())
