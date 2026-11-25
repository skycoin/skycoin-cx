package main

import (
	"github.com/skycoin/skycoin-cx/t0-types"
)

// llvmTemplate returns the LLVM IR an opcode of the LLVM target compiles to
// inline, decision D10, or "" for an opcode that calls its runtime function.
// A template moves one opcode from its runtime function to inline code, and
// changes nothing else.
//
// {a} and {b} are the loaded operands and {r} the result. {T} is the operand
// type, {R} the result type, {W} the bit width and {W1} one less, {F} f32 or
// f64 in a float intrinsic, and {t1} to {t9} fresh temporaries. A line
// "check {tN} DivideByZero" stops the run as the vm does when {tN} is true.
// Each template computes what t3-cx-runtime-vm computes, with Go semantics
// where LLVM leaves a case undefined or poison.
func llvmTemplate(row llvmRow) string {
	operand, operation := row.operand, row.operation
	if row.conversion {
		return llvmConversionTemplate(operand, row.result)
	}
	if operation == "print" {
		return ""
	}
	if operand == types.BOOL {
		switch operation {
		case "not":
			return "{t1} = icmp eq i8 {a}, 0\n{r} = zext i1 {t1} to i8"
		case "and", "or":
			return "{t1} = icmp ne i8 {a}, 0\n{t2} = icmp ne i8 {b}, 0\n{t3} = " + operation + " i1 {t1}, {t2}\n{r} = zext i1 {t3} to i8"
		case "eq", "uneq":
			predicate := map[string]string{"eq": "eq", "uneq": "ne"}[operation]
			return "{t1} = icmp ne i8 {a}, 0\n{t2} = icmp ne i8 {b}, 0\n{t3} = icmp " + predicate + " i1 {t1}, {t2}\n{r} = zext i1 {t3} to i8"
		}
		return ""
	}
	if operand == types.F32 || operand == types.F64 {
		switch operation {
		case "add", "sub", "mul", "div":
			return "{r} = f" + operation + " {T} {a}, {b}"
		case "neg":
			return "{r} = fneg {T} {a}"
		case "abs":
			return "{r} = call {T} @llvm.fabs.{F}({T} {a})"
		case "eq", "uneq", "lt", "lteq", "gt", "gteq":
			predicate := map[string]string{"eq": "oeq", "uneq": "une", "lt": "olt", "lteq": "ole", "gt": "ogt", "gteq": "oge"}[operation]
			return "{t1} = fcmp " + predicate + " {T} {a}, {b}\n{r} = zext i1 {t1} to i8"
		}
		// mod, max and min follow math.Mod, math.Max and math.Min, whose NaN,
		// infinity and signed zero cases differ from frem and llvm.maximum.
		return ""
	}
	signed := operand == types.I8 || operand == types.I16 || operand == types.I32 || operand == types.I64
	switch operation {
	case "add", "sub", "mul":
		return "{r} = " + operation + " {T} {a}, {b}"
	case "div":
		if signed {
			// Go defines MinInt / -1 as MinInt, LLVM sdiv makes it undefined.
			return "{t1} = icmp eq {T} {b}, 0\ncheck {t1} DivideByZero\n{t2} = icmp eq {T} {b}, -1\n{t3} = select i1 {t2}, {T} 1, {T} {b}\n{t4} = sdiv {T} {a}, {t3}\n{t5} = sub {T} 0, {a}\n{r} = select i1 {t2}, {T} {t5}, {T} {t4}"
		}
		return "{t1} = icmp eq {T} {b}, 0\ncheck {t1} DivideByZero\n{r} = udiv {T} {a}, {b}"
	case "mod":
		if signed {
			return "{t1} = icmp eq {T} {b}, 0\ncheck {t1} DivideByZero\n{t2} = icmp eq {T} {b}, -1\n{t3} = select i1 {t2}, {T} 1, {T} {b}\n{r} = srem {T} {a}, {t3}"
		}
		return "{t1} = icmp eq {T} {b}, 0\ncheck {t1} DivideByZero\n{r} = urem {T} {a}, {b}"
	case "neg":
		return "{r} = sub {T} 0, {a}"
	case "abs":
		return "{r} = call {T} @llvm.abs.{T}({T} {a}, i1 false)"
	case "max", "min":
		prefix := "u"
		if signed {
			prefix = "s"
		}
		return "{r} = call {T} @llvm." + prefix + operation + ".{T}({T} {a}, {T} {b})"
	case "eq", "uneq", "lt", "lteq", "gt", "gteq":
		predicates := map[string]string{"eq": "eq", "uneq": "ne", "lt": "ult", "lteq": "ule", "gt": "ugt", "gteq": "uge"}
		if signed {
			predicates = map[string]string{"eq": "eq", "uneq": "ne", "lt": "slt", "lteq": "sle", "gt": "sgt", "gteq": "sge"}
		}
		return "{t1} = icmp " + predicates[operation] + " {T} {a}, {b}\n{r} = zext i1 {t1} to i8"
	case "bitand", "bitor", "bitxor":
		return "{r} = " + operation[3:] + " {T} {a}, {b}"
	case "bitclear":
		return "{t1} = xor {T} {b}, -1\n{r} = and {T} {a}, {t1}"
	case "bitshl":
		// Go shifts by the count as unsigned, and a count of the width or
		// more gives 0. LLVM makes such a shift poison, so a select picks 0.
		return "{t1} = icmp uge {T} {b}, {W}\n{t2} = shl {T} {a}, {b}\n{r} = select i1 {t1}, {T} 0, {T} {t2}"
	case "bitshr":
		if signed {
			return "{t1} = icmp uge {T} {b}, {W}\n{t2} = select i1 {t1}, {T} {W1}, {T} {b}\n{r} = ashr {T} {a}, {t2}"
		}
		return "{t1} = icmp uge {T} {b}, {W}\n{t2} = lshr {T} {a}, {b}\n{r} = select i1 {t1}, {T} 0, {T} {t2}"
	}
	return ""
}

// llvmConversionTemplate converts a value of type from to type to.
func llvmConversionTemplate(from, to types.Code) string {
	float := func(code types.Code) bool { return code == types.F32 || code == types.F64 }
	signed := func(code types.Code) bool {
		return code == types.I8 || code == types.I16 || code == types.I32 || code == types.I64
	}
	switch {
	case float(from) && float(to):
		if from == types.F32 {
			return "{r} = fpext float {a} to double"
		}
		return "{r} = fptrunc double {a} to float"
	case float(from):
		// Go leaves an out of range conversion to the platform. freeze turns
		// the LLVM poison into some value, which is no stronger.
		if signed(to) {
			return "{t1} = fptosi {T} {a} to {R}\n{r} = freeze {R} {t1}"
		}
		return "{t1} = fptoui {T} {a} to {R}\n{r} = freeze {R} {t1}"
	case float(to):
		if signed(from) {
			return "{r} = sitofp {T} {a} to {R}"
		}
		return "{r} = uitofp {T} {a} to {R}"
	}
	switch fromSize, toSize := from.Size(), to.Size(); {
	case fromSize == toSize:
		return "{r} = add {T} {a}, 0"
	case fromSize > toSize:
		return "{r} = trunc {T} {a} to {R}"
	case signed(from):
		return "{r} = sext {T} {a} to {R}"
	}
	return "{r} = zext {T} {a} to {R}"
}
