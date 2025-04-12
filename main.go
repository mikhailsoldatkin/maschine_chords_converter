package main

import (
	"crypto/rand"
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
)

type Chord struct {
	Name  string `json:"name"`
	Notes []int  `json:"notes"`
}

type ChordSet struct {
	Chords  []Chord `json:"chords"`
	Name    string  `json:"name"`
	TypeID  string  `json:"typeId"`
	UUID    string  `json:"uuid"`
	Version string  `json:"version"`
}

const (
	version             = "1.0.0"
	midiExtension       = ".mid"
	baseChordName       = "Chd" // for empty chords
	baseNote            = 60    // C3
	maxSetFolderNameLen = 10    // maximum length of set folder name
	minChordNumber      = 1
	maxChordNumber      = 12
	maxSetNumber        = 16
	debug               = false
)

var setNumber = 1

// getExecutableDir returns the directory of the executable file.
func getExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error determining executable path:", err)
		return ""
	}
	return filepath.Dir(exePath)
}

func main() {
	exeDir := getExecutableDir()
	setsFolder := filepath.Join(exeDir, "sets")

	if debug {
		setsFolder = "./sets"
	}

	if err := filepath.WalkDir(setsFolder, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dir.IsDir() && path != setsFolder && len(dir.Name()) <= maxSetFolderNameLen {
			processChordSet(path, dir.Name(), setsFolder)
		}

		return nil
	}); err != nil {
		fmt.Println("Directory traversal error:", err)
	}

	if debug {
		fmt.Println("Processing complete.")
		return
	}

	fmt.Println("Processing complete. Press Enter to exit...")
	_, _ = fmt.Scanln()
}

func processChordSet(path, currentSet, setsFolder string) {
	fmt.Println("Processing set:", currentSet)

	if setNumber > maxSetNumber {
		return
	}

	chords := make([]Chord, maxChordNumber)
	for i := range chords {
		chords[i] = Chord{
			Name:  fmt.Sprintf("%s %d", baseChordName, i+1),
			Notes: []int{},
		}
	}

	if err := filepath.WalkDir(path, func(midiPath string, f fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() || !strings.HasSuffix(f.Name(), midiExtension) {
			return nil
		}

		chordNumber, chordName := parseMidiFileName(f.Name())
		if chordNumber < minChordNumber || chordNumber > maxChordNumber || chordNumber == 0 || chordName == "" {
			return nil
		}

		notes := readMidiNotes(midiPath)
		slices.Sort(notes)

		chords[chordNumber-1] = Chord{
			Name:  chordName,
			Notes: notes,
		}

		return nil
	}); err != nil {
		fmt.Printf("Error processing MIDI files: %v\n", err)
		return
	}

	output := ChordSet{
		Chords:  chords,
		Name:    currentSet,
		UUID:    generateUUID(),
		TypeID:  "native-instruments-chord-set",
		Version: version,
	}

	jsonData, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		fmt.Printf("Error marshaling JSON for %s: %v\n", path, err)
		return
	}

	outFile := filepath.Join(filepath.Dir(setsFolder), fmt.Sprintf("user_chord_set_0%d.json", setNumber))
	if err = os.WriteFile(outFile, jsonData, 0644); err != nil {
		fmt.Printf("Error writing JSON file %s: %v\n", outFile, err)
		return
	}

	fmt.Println("Generated file:", outFile)

	setNumber++
}

// parseMidiFileName parses and checks midi file name.
// midi file name pattern should be: "12 Amin9.mid" or "1 Cmin.mid" (number (1 or 2 digits) => 1 space => chord name => dot => mid)
func parseMidiFileName(fileName string) (int, string) {
	re := regexp.MustCompile(`^(\d{1,2}) (.+?)\.mid$`)
	match := re.FindStringSubmatch(fileName)
	if len(match) != 3 {
		return 0, ""
	}

	index, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, ""
	}

	return index, strings.TrimSpace(match[2])
}

func readMidiNotes(path string) []int {
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
		fmt.Printf("Failed to read MIDI file %s, error: %s", path, err)
	}

	var relative []int
	for _, note := range notes {
		relative = append(relative, note-baseNote)
	}

	return relative
}

func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
