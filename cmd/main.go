package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/reader"
)

const (
	version             = "1.0.0"
	midiExtension       = ".mid"
	baseChordName       = "Chd" // for empty chords
	baseNote            = 60    // C3
	maxSetFolderNameLen = 10    // maximum length of set folder name
	minChordNumber      = 1
	maxChordNumber      = 12
	maxSetNumber        = 16
	debug               = true
)

var re = regexp.MustCompile(`^(\d{1,2}) (.+?)\.mid$`)

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

type Converter struct {
	ChordSets  []ChordSet
	SetCount   uint
	SetsFolder string
	Debug      bool
}

func main() {
	converter := Converter{
		ChordSets: []ChordSet{},
		Debug:     debug,
	}

	if err := converter.Run(); err != nil {
		log.Fatal(err.Error())
	}

	if converter.Debug {
		fmt.Println("processing complete...")
		return
	}

	fmt.Println("processing complete. Press Enter to exit...")
	_, _ = fmt.Scanln()
}

func (c *Converter) Run() error {
	if err := c.getExecutableDir(); err != nil {
		return err
	}

	if err := c.processChordSetFolders(); err != nil {
		return err
	}

	if err := c.outputJsonFiles(); err != nil {
		return err
	}

	return nil
}

func (c *Converter) processChordSetFolders() error {
	if err := filepath.WalkDir(c.SetsFolder, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dir.IsDir() && path != c.SetsFolder && len(dir.Name()) <= maxSetFolderNameLen {
			if err = c.processOneChordSetFolder(path, dir.Name()); err != nil {
				return fmt.Errorf("error processing folder %s: %w", path, err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("directory traversal error: %w", err)
	}

	return nil
}

func (c *Converter) getExecutableDir() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error determining executable path: %w", err)

	}

	c.SetsFolder = filepath.Dir(execPath)

	if c.Debug {
		c.SetsFolder = "./sets"
	}

	return nil
}

func (c *Converter) processOneChordSetFolder(setPath, setName string) error {
	fmt.Printf("processing set: %s\n", setName)

	if c.SetCount >= maxSetNumber {
		return nil
	}

	chords := make([]Chord, maxChordNumber)
	for i := range chords {
		chords[i] = Chord{
			Name:  fmt.Sprintf("%s %d", baseChordName, i+1),
			Notes: []int{},
		}
	}

	if err := filepath.WalkDir(setPath, func(chordPath string, file fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if file.IsDir() || !strings.HasSuffix(file.Name(), midiExtension) {
			return nil
		}

		chordNumber, chordName, err := c.parseMidiFileName(file.Name())
		if err != nil {
			return err
		}

		if chordNumber < minChordNumber || chordNumber > maxChordNumber || chordNumber == 0 || chordName == "" {
			return nil // TODO ?
		}

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

	c.ChordSets = append(c.ChordSets, ChordSet{
		Chords:  chords,
		Name:    setName,
		UUID:    generateUUID(),
		TypeID:  "native-instruments-chord-set",
		Version: version,
	})

	c.SetCount++

	return nil
}

func (c *Converter) parseMidiFileName(fileName string) (int, string, error) {
	match := re.FindStringSubmatch(fileName)
	if len(match) != 3 {
		return 0, "", fmt.Errorf("invalid file name: %s", fileName)
	}

	index, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, "", fmt.Errorf("can't convert chord number to integer: %s", fileName)
	}

	return index, strings.TrimSpace(match[2]), nil
}

func (c *Converter) outputJsonFiles() error {
	for i, chordSet := range c.ChordSets {
		jsonData, err := json.MarshalIndent(chordSet, "", "    ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON for %s: %w", chordSet.Name, err)
		}

		outFile := filepath.Join(filepath.Dir(c.SetsFolder), fmt.Sprintf("user_chord_set_0%d.json", i+1))
		if err = os.WriteFile(outFile, jsonData, 0644); err != nil {
			return fmt.Errorf("error writing JSON file %s: %w", outFile, err)
		}

		fmt.Println("generated file:", outFile)
	}

	return nil
}

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

func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
