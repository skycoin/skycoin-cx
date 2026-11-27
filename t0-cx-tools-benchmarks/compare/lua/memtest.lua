-- Address in address, walking ones and moving inversions over 1048576 ui32.
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

local MEM_N = 1048576
local MEM_WALK = 65536

local function main()
	local h = FNV_OFFSET
	local allbits = 0xFFFFFFFF
	local mem = {}
	local n = MEM_N

	local bad = 0
	local sum = 0
	for i = 0, n - 1 do
		mem[i + 1] = i
	end
	for i = 0, n - 1 do
		local read = mem[i + 1]
		if read ~= i then
			bad = bad + 1
		end
		sum = sum + read
	end
	for i = 0, n - 1 do
		mem[i + 1] = i ~ allbits
	end
	for i = 0, n - 1 do
		local read = mem[i + 1]
		if read ~= (i ~ allbits) then
			bad = bad + 1
		end
		sum = sum + read
	end
	h = fnv(h, bad)
	h = fnv(h, sum)
	local total = bad

	for bit = 0, 31 do
		local pattern = 1 << bit
		bad = 0
		sum = 0
		for i = 1, MEM_WALK do
			mem[i] = pattern
		end
		for i = 1, MEM_WALK do
			local read = mem[i]
			if read ~= pattern then
				bad = bad + 1
			end
			sum = sum + read
		end
		h = fnv(h, bad)
		h = fnv(h, sum)
		total = total + bad
	end

	for _, pattern in ipairs({ 0x00000000, 0xFFFFFFFF, 0x55555555, 0xAAAAAAAA }) do
		local inverse = pattern ~ allbits
		bad = 0
		sum = 0
		for i = 1, n do
			mem[i] = pattern
		end
		for i = 1, n do
			local read = mem[i]
			if read ~= pattern then
				bad = bad + 1
			end
			sum = sum + read
			mem[i] = inverse
		end
		for i = n, 1, -1 do
			local read = mem[i]
			if read ~= inverse then
				bad = bad + 1
			end
			sum = sum + read
			mem[i] = pattern
		end
		for i = 1, n do
			local read = mem[i]
			if read ~= pattern then
				bad = bad + 1
			end
			sum = sum + read
		end
		h = fnv(h, bad)
		h = fnv(h, sum)
		total = total + bad
	end
	return report("memtest", h, total == 0)
end

os.exit(main())
