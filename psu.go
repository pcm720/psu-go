package psu

import (
	"encoding/binary"
	"io"
	"time"
)

// Based on research by McCaulay Hudson
// https://mccaulay.co.uk/mast1c0re-part-1-modifying-ps2-game-save-files/

type PSUHeader struct {
	Type     PSUType
	_        [2]byte // always 0
	Size     uint32  // Number of entries for directory, number of bytes for filesize
	Created  PSUTimestamp
	_        [8]byte // Some bytes are used by EMS, but not needed for our use case
	Modified PSUTimestamp
	_        [32]byte // Some bytes are used by EMS, but not needed for our use case
	Name     [32]byte
	_        [416]byte
}

type PSUType uint16

const (
	PSUType_Directory = 0x8427
	PSUType_File      = 0x8497
)

type PSUTimestamp struct {
	Zero    uint8
	Seconds uint8
	Minutes uint8
	Hours   uint8
	Day     uint8
	Month   uint8
	Year    uint16
}

type File struct {
	Name     string
	Created  time.Time
	Modified time.Time
	Data     []byte
}

// Creates PSU file with given root directory name and files
func BuildPSU(b io.Writer, rootDirName string, files []File) error {
	p := PSUHeader{
		Type:     PSUType_Directory,
		Size:     uint32(3 + len(files)),
		Created:  newPSUTimestamp(time.Now()),
		Modified: newPSUTimestamp(time.Now()),
	}
	copy(p.Name[:], []byte(rootDirName))

	if err := binary.Write(b, binary.LittleEndian, p); err != nil {
		return err
	}

	p.Size = 0
	p.Name = [32]byte{}
	copy(p.Name[:], ".")
	if err := binary.Write(b, binary.LittleEndian, p); err != nil {
		return err
	}

	copy(p.Name[:], "..")
	if err := binary.Write(b, binary.LittleEndian, p); err != nil {
		return err
	}

	// Embed files
	for _, f := range files {
		if err := writeFile(b, f); err != nil {
			return err
		}
	}

	return nil
}

// Creates PSU timestamp from Time
func newPSUTimestamp(t time.Time) PSUTimestamp {
	t = t.UTC()
	return PSUTimestamp{
		Seconds: uint8(t.Second()),
		Minutes: uint8(t.Minute()),
		Hours:   uint8(t.Hour()),
		Day:     uint8(t.Day()),
		Month:   uint8(t.Month()),
		Year:    uint16(t.Year()),
	}
}

// Used to pad file data
var padding []byte

// Writes PSU header and file data to passed Writer
func writeFile(b io.Writer, f File) error {
	p := PSUHeader{
		Type:     PSUType_File,
		Size:     uint32(len(f.Data)),
		Created:  newPSUTimestamp(f.Created),
		Modified: newPSUTimestamp(f.Modified),
	}
	copy(p.Name[:], []byte(f.Name))
	if err := binary.Write(b, binary.LittleEndian, p); err != nil {
		return err
	}

	// Ensure file data is multiple of 1024
	padSize := 1024 - (len(f.Data) % 1024)
	if (padSize) > 0 {
		if len(padding) < padSize {
			padding = make([]byte, padSize)
		}
	}

	// Write file data
	if err := binary.Write(b, binary.LittleEndian, f.Data); err != nil {
		return err
	}
	// Write padding
	if err := binary.Write(b, binary.LittleEndian, padding[:padSize]); err != nil {
		return err
	}

	return nil
}
