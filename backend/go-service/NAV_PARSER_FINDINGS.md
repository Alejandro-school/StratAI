# NavMesh Parser Findings (v35)

## 1. Mixed Endianness
The Source 2 NavMesh (v35) appears to use **Mixed Endianness**:
- **Area 0**: Uses **Little Endian** for all fields.
- **Area 1+**: Uses **Big Endian** for `ID`, `Flags`, `Hull`, `Poly`, `Unk`.
- **Variable Length Fields**: `ConnCount`, `LadderCount`, etc. seem to follow the Area's endianness (or are consistently LE/BE, still debugging `ConnCount`).

## 2. Alignment Issue
There is a **4-byte alignment issue** after reading Area 0.
- The parser was reading `ID=0` for Area 1 because it was reading the last 4 bytes of `Time` (zeros) as `ID`.
- **Fix**: Reading 4 dummy bytes after Area 0 aligns the stream correctly to the start of Area 1.

## 3. Area 1 Header Parsed
With the alignment fix and Big Endian reading, Area 1 header is parsed as:
- `ID`: 256 (0x0100) - Likely ID 1 in Big Endian (if high bytes ignored) or just 256.
- `Flags`: 0
- `Hull`: 256 (0x0100)
- `Poly`: 2048 (0x0800)
- `Unk`: 256 (0x0100)

These values are consistent and valid (unlike the huge numbers from Little Endian interpretation).

## 4. Place Names
- `scan_nav_strings.py` confirmed that **Place Names (strings) are NOT present** in the `.nav` file.
- We successfully read `PlaceID` (integer) for each area.
- **Next Step**: We need to find where `PlaceID` maps to strings (e.g., `resource/csgo_english.txt` or a separate `.kv3` file).

## 5. Remaining Blocker
- `ConnCount` for Area 1 is read as `65536` (0x10000).
- This suggests either:
    - `ConnCount` is Little Endian (reading `00 00 01 00` as 65536).
    - Padding is different for Area 1.
    - We are still slightly misaligned for the variable data section.
