# Maschine Chords Converter

**Maschine Chords Converter** is a utility for generating JSON files containing chord sets based on MIDI files.  
These files then can be loaded into the Maschine 3.0 application from Native Instruments. Utility is implemented using
Go.

## Table of contents

- [Folder Structure](#folder-structure)
- [Naming Format for Chord Set Folders](#naming-format-for-chord-set-folders)
- [Naming Format for MIDI Files](#naming-format-for-midi-files)
- [Processing Procedure](#processing-procedure)
- [Output Files](#output-files)
- [How to Run the Utility](#how-to-run-the-utility)

## Folder structure

The utility expects the following structure:

```
├── maschine_chords_converter (utility file)
└── sets/ (folder with chord sets)
├── Set 1/ (folder with MIDI chord files)
│ ├── 1 Cmin.mid (midi file with one chord)
│ ├── 2 Dmin.mid
│ └── ...
├── Set 2/
│ ├── 1 Emaj.mid
│ └── ...
└── ...
```

## Naming format for chord set folders

The folder name for each chord set must not exceed 10 characters. On the Maschine display, only 10 characters of
the chord set name are shown.  
*Note: Maschine "shortens" names if their length exceeds the limit (e.g., "Very Long Set Name" becomes "VryLngSt").*

**Examples:**

- `set 1 Cm`
- `set F#m`
- `house Gm`

## Naming format for MIDI files

Within each chord set folder, MIDI files must be named in the following format:

`<number><space><chord name>.mid`

**Examples:**

- `1 Amin.mid`
- `01 Cmaj.mid`
- `12 F#min9.mid`

// TODO: The chord name is currently not restricted; verify the maximum length.

**Format Requirements:**

- `<number>` — a number consisting of 1 or 2 digits, and must be in the range from 1 to 12 (inclusive).
- There must be exactly 1 space between the number and the chord name.
- The chord name must not be empty.
- The file extension must be `.mid`.

If a file does not meet this format (e.g., the number is out of range or the formatting is incorrect), it will be
skipped.

## Processing procedure

1. **Folder Scanning:**  
   The utility begins by scanning the **sets** folder for all subfolders whose names do not exceed 10 characters.

2. **Processing Each Chord Set (Subfolder):**  
   For each subfolder found:
    - An array of 12 chords is created. If a MIDI file for a specific number is not found, a default empty chord with
      the name `Chd <number>` is created.
    - All files in the subfolder are scanned. Files with the `.mid` extension are processed according to the naming
      format.
    - Notes are extracted from each MIDI file, converted to the required values relative to the note C3, and sorted.
    - The corresponding chord in the array is replaced with the data obtained from the file.

3. **JSON File Generation:**  
   After processing the chord set, a JSON file is generated containing:
    - A list of chords with their names and note arrays.
    - The chord set name (subfolder name).
    - A UUID generated randomly.
    - The chord set type (`native-instruments-chord-set`).
    - The version (currently set to "1.0.0").

## Output files

- The generated JSON files are saved in the parent directory relative to the **sets** folder.
- The file name is generated using the following pattern:  
  `user_chord_set_0X.json`  
  where **X** is the sequential number of the processed chord set.
- The utility will create up to 16 JSON files (the limit is defined in the code by the constant `maxSetNumber`).

## How to run the utility

1. Create a folder named **"sets"** anywhere on your computer.
2. Inside the **sets** folder, create up to 16 subfolders with chord sets following
   the [naming rules for chord sets](#naming-format-for-chord-set-folders).
3. In each subfolder, place up to 12 MIDI files following
   the [naming rules for MIDI files](#naming-format-for-midi-files).
4. Place the utility file **maschine_chords_converter** in the same folder as the **sets** folder.
5. Run the utility by double-clicking the file.
6. The utility will start processing and display messages in the console:
    - Messages indicating the processing of each chord set.
    - Error messages for any files that do not match the format.
7. Upon completion, the console will display the message: "Processing complete. Press Enter to exit...". Press Enter to
   close the program.
8. Copy the generated JSON files to the following path:
    - **Mac:** `/Users/username/Library/Application Support/Native Instruments/Shared/User Chords`
    - **Windows:** `C:\Users\username\AppData\Local\Native Instruments\Shared\User Chords\`
9. Open Maschine 3.0, load the chord sets, and start creating music.

## Platform limitations

The program has been tested only on MacOS. Using on other operating systems is not guaranteed.

## Authors and notes

**Maschine Chords Converter** is created by Mikhail Soldatkin (c) 2025.  
**Maschine** is a trademark of **Native Instruments**.

## You can follow me on

[Spotify](https://open.spotify.com/artist/5y9uI0PQYtxPYZEL3X88JR)  
[Bandcamp](https://inchange.bandcamp.com/)  
