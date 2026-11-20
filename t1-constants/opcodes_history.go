package constants

// Opcode lifecycle events in version order, see opcodes.go for the conventions.
//
// version     event         number name
// 0000000000  assigned      every row of OpcodeLifecycleTable with version added 0
// 2611200001  assigned      586 str.equal_by_value
// 2611200001  assigned      587 str.equal_by_pointer
// 2611200001  assigned      588 str.compare_by_value
// 2611200001  assigned      589 ptr.eq
// 2611200001  assigned      590 ptr.uneq
// 2611200001  deprecated    559 str.eq
// 2611200001  deprecated    560 str.uneq
// 2611200001  deprecated    582 str.lt
// 2611200001  deprecated    583 str.lteq
// 2611200001  deprecated    584 str.gt
// 2611200001  deprecated    585 str.gteq
// 2711200001  reassignable  559 str.eq
// 2711200001  reassignable  560 str.uneq
// 2711200001  reassignable  582 str.lt
// 2711200001  reassignable  583 str.lteq
// 2711200001  reassignable  584 str.gt
// 2711200001  reassignable  585 str.gteq
//
// Interval format. A version is yymmddnnnn, so version / 100000000 is the
// year since 2000, the next two digits the month, the next two the day and
// the last four the release on that day. The reassignable version of a
// deprecation is the same month and day OpcodeReassignmentIntervalYears years
// later, release 0001. Deprecated 2611200001 (2026-11-20 release 0001) is
// reassignable from 2711200001 (2027-11-20 release 0001). A 29 February
// deprecation without a 29 February in the target year moves to 1 March.
// TestOpcodeLifecycleHistoryMatchesTable fails when this list and the table
// differ.

// OpcodeReassignmentIntervalYears is how long a deprecated opcode number
// stays unassigned.
const OpcodeReassignmentIntervalYears = 1

// LanguageVersionReassignable returns the version from which an opcode number
// deprecated in version deprecated may be assigned again.
func LanguageVersionReassignable(deprecated int64) int64 {
	year, month, day := languageVersionDate(deprecated)
	return languageVersionFromDate(year+OpcodeReassignmentIntervalYears, month, day, 1)
}

// LanguageVersionValid reports whether version is a yymmddnnnn with a real
// date and a release number of at least 1.
func LanguageVersionValid(version int64) bool {
	year, month, day := languageVersionDate(version)
	release := version % 10000
	if version <= 0 || version > 9912319999 || release < 1 || month < 1 || month > 12 || day < 1 {
		return false
	}
	return day <= languageVersionDaysInMonth(year, month)
}

func languageVersionDate(version int64) (year, month, day int) {
	return int(version / 100000000), int(version / 1000000 % 100), int(version / 10000 % 100)
}

func languageVersionFromDate(year, month, day int, release int64) int64 {
	if day > languageVersionDaysInMonth(year, month) {
		month, day = month+1, 1
	}
	return int64(year)*100000000 + int64(month)*1000000 + int64(day)*10000 + release
}

func languageVersionDaysInMonth(year, month int) int {
	switch month {
	case 2:
		if (2000+year)%4 == 0 && ((2000+year)%100 != 0 || (2000+year)%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}
