# Student ID Format Samples

## Overview
Each student will have a unique ID that encodes their details. Students can login using either this ID or their email address.

## Proposed Format Options

### Option 1: Comprehensive Format (Recommended)
**Format:** `[YEAR][COURSE_CODE][BRANCH_CODE][DISTRICT_CODE][SEQUENCE][CHECKSUM]`

**Structure:**
- `YEAR`: 4 digits (enrollment year, e.g., 2024)
- `COURSE_CODE`: 3-4 uppercase letters (course abbreviation, e.g., CS for Computer Science)
- `BRANCH_CODE`: 2-3 uppercase letters (branch abbreviation, e.g., UG for Undergraduate, or 00 if no branch)
- `DISTRICT_CODE`: 3 uppercase letters (first 3 letters of district, e.g., KMP for Kampala)
- `SEQUENCE`: 4 digits (sequential number within course+branch+year, e.g., 0001)
- `CHECKSUM`: 1 digit (Luhn algorithm checksum for validation)

**Examples:**
```
2024CS-UG-KMP-0001-5
2024ENG-MA-JIN-0023-8
2024MED-PG-ENT-0156-2
2024LAW-UG-WAK-0001-9
```

**Pros:**
- Highly readable and meaningful
- Encodes course, branch, year, district, and sequence
- Includes checksum for validation
- Easy to parse and understand

**Cons:**
- Longer format (18-20 characters)
- Requires course and branch to have codes/abbreviations

---

### Option 2: Compact Numeric Format
**Format:** `[YEAR][COURSE_ID][BRANCH_ID][SEQUENCE][CHECKSUM]`

**Structure:**
- `YEAR`: 2 digits (last 2 digits of enrollment year, e.g., 24 for 2024)
- `COURSE_ID`: 3 digits (padded course ID, e.g., 001, 042)
- `BRANCH_ID`: 2 digits (padded branch ID, e.g., 01, 00 if no branch)
- `SEQUENCE`: 4 digits (sequential number, e.g., 0001)
- `CHECKSUM`: 1 digit (Luhn algorithm checksum)

**Examples:**
```
24001010001-5  (2024, Course 1, Branch 1, Sequence 1)
24042020023-8  (2024, Course 42, Branch 2, Sequence 23)
24001000156-2  (2024, Course 1, No branch, Sequence 156)
```

**Pros:**
- Compact (12-13 characters)
- Works with numeric IDs
- Easy to generate programmatically

**Cons:**
- Less human-readable
- Requires lookup to understand course/branch

---

### Option 3: Hybrid Alphanumeric Format
**Format:** `[COURSE_CODE]-[YEAR]-[BRANCH_CODE]-[SEQUENCE]-[CHECKSUM]`

**Structure:**
- `COURSE_CODE`: 3-4 uppercase letters (course abbreviation)
- `YEAR`: 2 digits (last 2 digits of enrollment year)
- `BRANCH_CODE`: 2-3 uppercase letters or "00" if no branch
- `SEQUENCE`: 4 digits (sequential number)
- `CHECKSUM`: 1 digit (Luhn checksum)

**Examples:**
```
CS-24-UG-0001-5
ENG-24-MA-0023-8
MED-24-PG-0156-2
LAW-24-UG-0001-9
```

**Pros:**
- Balanced readability and compactness
- Easy to read course and branch
- Moderate length (15-17 characters)

**Cons:**
- Requires course/branch codes
- Hyphens may need to be handled in URLs

---

### Option 4: Base36 Encoded Format
**Format:** `[YEAR][COURSE_BASE36][BRANCH_BASE36][SEQUENCE_BASE36][CHECKSUM]`

**Structure:**
- `YEAR`: 2 digits (last 2 digits)
- `COURSE_BASE36`: 2 characters (course ID in base36, e.g., 01, 0K)
- `BRANCH_BASE36`: 2 characters (branch ID in base36, e.g., 01, 00)
- `SEQUENCE_BASE36`: 3 characters (sequence in base36, e.g., 001, 0A1)
- `CHECKSUM`: 1 character (base36 checksum)

**Examples:**
```
2401010001-5
240K0200A1-8
2401000156-2
```

**Pros:**
- Very compact (10-11 characters)
- Uses alphanumeric encoding

**Cons:**
- Less readable
- Requires base36 conversion logic

---

## Recommended: Option 1 (Comprehensive Format)

**Final Format:** `[YEAR][COURSE_CODE]-[BRANCH_CODE]-[DISTRICT_CODE]-[SEQUENCE]-[CHECKSUM]`

**Example IDs:**
```
2024CS-UG-KMP-0001-5
2024ENG-MA-JIN-0023-8
2024MED-PG-ENT-0156-2
2024LAW-UG-WAK-0001-9
2024CS-UG-KMP-0002-3
```

**Implementation Notes:**
1. Course codes should be standardized (e.g., CS, ENG, MED, LAW)
2. Branch codes should be standardized (e.g., UG, PG, MA, PhD)
3. District codes use first 3 uppercase letters
4. Sequence resets each year per course+branch combination
5. Checksum uses Luhn algorithm for validation
6. If branch is null, use "00" or "NA"
7. If district is empty, use "XXX" or first 3 letters of course name

---

## Alternative: Simplified Format (If codes are not available)

**Format:** `STU-[YEAR]-[COURSE_ID]-[SEQUENCE]-[CHECKSUM]`

**Examples:**
```
STU-2024-001-0001-5
STU-2024-042-0023-8
STU-2024-001-0156-2
```

This format works even if course/branch codes are not standardized yet.

---

## Next Steps
1. Choose a format
2. Add `StudentID` field to Student model
3. Create ID generation function
4. Update login to accept ID or email
5. Add validation and uniqueness checks

