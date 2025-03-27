// Code based off github.com/mccoyst/ogg
// and github.com/diamondburned/oggreader

package main

import (
	"bytes"
	"fmt"
	"io"
)

const (
	headerSize     = 27
	maxSegmentSize = 255
	maxPacketSize  = maxSegmentSize * 255
	maxPageSize    = headerSize + maxSegmentSize + maxPacketSize
)

type Decoder struct {
	r            io.Reader
	buffer       [maxPageSize]byte
	header       []byte
	segmentTable []byte
	data         []byte
	nSegs        int
	iBuffer      int
	iSegTable    int
}

var oggs = [...]byte{'O', 'g', 'g', 'S'}

func NewDecoder(r io.Reader) *Decoder {
	d := &Decoder{r: r, iBuffer: 0, iSegTable: 0}
	d.readPage()

	return d
}

func (d *Decoder) readPage() error {
	d.header = d.buffer[:headerSize]

	_, err := io.ReadFull(d.r, d.header)
	if err != nil {
		return err
	}

	if !bytes.Equal(d.header[:4], oggs[:]) {
		return fmt.Errorf("Invalid ogg header")
	}

	d.nSegs = int(d.header[26])

	d.segmentTable = d.buffer[headerSize : headerSize+d.nSegs]
	_, err = io.ReadFull(d.r, d.segmentTable)
	if err != nil {
		return err
	}

	dataSize := 0
	for _, v := range d.segmentTable {
		dataSize += int(v)
	}

	d.data = d.buffer[headerSize+d.nSegs : headerSize+d.nSegs+dataSize]
	_, err = io.ReadFull(d.r, d.data)
	if err != nil {
		return err
	}

	return nil
}

func (d *Decoder) GetPacket() ([]byte, error) {
	var packet []byte

	for {
		if d.iSegTable >= d.nSegs {
			err := d.readPage()
			if err != nil {
				return nil, err
			}
			d.iSegTable = 0
			d.iBuffer = 0
		}

		segSize := int(d.segmentTable[d.iSegTable])
		d.iSegTable++

		packet = append(packet, d.data[d.iBuffer:d.iBuffer+segSize]...)
		d.iBuffer += segSize

		if segSize < 255 {
			break
		}
	}

	return packet, nil
}
