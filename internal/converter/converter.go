package converter

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/reader"

	"maschine_chords_converter/internal/helpers"
)

const (
	version             = "1.0.0" // defines the current version of the chord sets
	midiExtension       = ".mid"  // defines the required extension for MIDI files
	baseChordName       = "Chd"   // is used for creating a default chord name (for empty chords)
	baseNote            = 60      // the base note (C3) relative to which the note values will be calculated
	maxSetFolderNameLen = 10      // defines the maximum length for a chord set folder name
	minChordNumber      = 1       // the minimum allowed chord number
	maxChordNumber      = 12      // the maximum allowed chord number (and, consequently, the number of chords in a set)
	maxSetNumber        = 16      // the maximum number of chord sets that can be processed
	setsFolderName      = "sets"  // folder name for chord sets
)

// re regular expression used to validate and parse MIDI file names. Expected file name format: "12 Amin9.mid" or "1 Cmin.mid"
var re = regexp.MustCompile(`^(\d{1,2}) (.+?)\.mid$`)

// Chord represents a single chord.
type Chord struct {
	Name  string `json:"name"`  // name of a chord
	Notes []int  `json:"notes"` // slice of chord notes
}

// ChordSet represents a set of chords along with properties required for generating a JSON file.
type ChordSet struct {
	Chords  []Chord `json:"chords"`  // slice of chords
	Name    string  `json:"name"`    // name of a set
	TypeID  string  `json:"typeId"`  // metadata
	UUID    string  `json:"uuid"`    // metadata
	Version string  `json:"version"` // metadata
}

// Converter converts MIDI files into JSON chord sets.
type Converter struct {
	chordSets  []ChordSet // processed chord sets
	setsFolder string     // path to the folder containing chord set directories
	debug      bool       // debug mode flag
}

// New creates and returns a new Converter instance.
func New() Converter {
	return Converter{chordSets: make([]ChordSet, 0, maxSetNumber)}
}

// SetDebug sets the debug mode of the Converter instance.
func (c *Converter) SetDebug(debug bool) {
	c.debug = debug
}

// Run performs the sequence of operations:
// 1. Determines the path for main folder with sets
// 2. Processes chord set folders in main folder
// 3. Outputs JSON files
func (c *Converter) Run() error {
	if err := c.getSetsFolder(); err != nil {
		return err
	}

	if err := c.processSetsFolder(); err != nil {
		return err
	}

	if err := c.outputJsonFiles(); err != nil {
		return err
	}

	return nil
}

// getSetsFolder determines the directory of the executable and sets the setsFolder path.
// If debug mode is enabled, it uses the local "./sets" directory.
func (c *Converter) getSetsFolder() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error determining executable path: %w", err)
	}

	c.setsFolder = filepath.Dir(execPath)

	if c.debug {
		c.setsFolder = setsFolderName
	}

	return nil
}

// processSetsFolder scans the setsFolder directory for subfolders with valid names and processes each of them as a chord set.
func (c *Converter) processSetsFolder() error {
	if err := filepath.WalkDir(c.setsFolder, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// only process directories that are not the root folder and whose names are within the allowed length
		if dir.IsDir() && path != c.setsFolder && len(dir.Name()) <= maxSetFolderNameLen {
			if dir.Name() == setsFolderName {
				return nil
			}

			if err = c.processOneSetFolder(path, dir.Name()); err != nil {
				return fmt.Errorf("error processing set folder %s: %w", path, err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("directory traversal error: %w", err)
	}

	return nil
}

// processOneSetFolder processes a single chord set folder.
// It reads MIDI files, parses their names, extracts note data, and builds a ChordSet structure.
func (c *Converter) processOneSetFolder(setPath, setName string) error {
	fmt.Printf("processing set: %s\n", setName)

	if len(c.chordSets) >= maxSetNumber {
		return nil
	}

	// initialize the chords array with default values.
	chords := make([]Chord, maxChordNumber)
	for i := range chords {
		chords[i] = Chord{
			Name:  fmt.Sprintf("%s %d", baseChordName, i+1),
			Notes: []int{},
		}
	}

	// walk through the files in the chord set folder.
	if err := filepath.WalkDir(setPath, func(chordPath string, file fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip directories and files without the .mid extension.
		if file.IsDir() || !strings.HasSuffix(file.Name(), midiExtension) {
			return nil
		}

		// parse the chord file name to extract the chord number and chord name.
		chordNumber, chordName, err := c.parseChordFileName(file.Name())
		if err != nil {
			return err
		}

		// skip the file if the chord number is out of range or if the chord name is empty.
		if chordNumber < minChordNumber || chordNumber > maxChordNumber || chordNumber == 0 || chordName == "" {
			return nil
		}

		// read the chord notes from the MIDI file.
		chordNotes, err := c.readChordNotes(chordPath)
		if err != nil {
			return err
		}
		slices.Sort(chordNotes)

		chords[chordNumber-1] = Chord{
			Name:  chordName,
			Notes: chordNotes,
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error processing set %s: %w", setName, err)
	}

	// append the processed chord set to the list.
	c.chordSets = append(c.chordSets, ChordSet{
		Chords:  chords,
		Name:    setName,
		UUID:    helpers.GenerateUUID(),
		TypeID:  "native-instruments-chord-set",
		Version: version,
	})

	return nil
}

// parseChordFileName parses a MIDI file name and extracts the chord number and chord name.
func (c *Converter) parseChordFileName(fileName string) (int, string, error) {
	match := re.FindStringSubmatch(fileName)
	if len(match) != re.NumSubexp()+1 { // ensure match length equals full match (1) + number of subexpressions (2)
		return 0, "", fmt.Errorf("invalid file name: %s", fileName)
	}

	number, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, "", fmt.Errorf("can't convert chord number to integer: %s", fileName)
	}

	name := strings.TrimSpace(match[2])

	return number, name, nil
}

// readChordNotes reads notes from a MIDI file and returns a slice of relative to baseNote note values.
func (c *Converter) readChordNotes(path string) ([]int, error) {
	var notes []int
	seen := make(map[int]bool)

	rd := reader.New(
		reader.NoLogger(),
		reader.NoteOn(func(pos *reader.Position, channel, key, vel uint8) {
			if vel > 0 && !seen[int(key)] {
				notes = append(notes, int(key))
				seen[int(key)] = true
			}
		}),
	)

	err := reader.ReadSMFFile(rd, path)
	if err != nil {
		return notes, fmt.Errorf("failed to read MIDI file %s: %w", path, err)
	}

	var relative []int
	for _, note := range notes {
		relative = append(relative, note-baseNote)
	}

	return relative, nil
}

// outputJsonFiles generates and saves JSON files for each processed chord set.
// The JSON files are saved one directory level above the setsFolder.
func (c *Converter) outputJsonFiles() error {
	for i, chordSet := range c.chordSets {
		jsonData, err := json.MarshalIndent(chordSet, "", "    ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON for %s: %w", chordSet.Name, err)
		}

		outFolder := c.setsFolder // same folder
		if c.debug {
			outFolder = filepath.Dir(c.setsFolder) // one level above
		}

		outFile := filepath.Join(outFolder, fmt.Sprintf("user_chord_set_0%d.json", i+1))
		if err = os.WriteFile(outFile, jsonData, 0644); err != nil {
			return fmt.Errorf("error writing JSON file %s: %w", outFile, err)
		}

		fmt.Println("generated file:", outFile)
	}

	return nil
}
