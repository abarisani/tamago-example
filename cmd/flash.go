// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build usbarmory

package cmd

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/usbarmory/tamago-example/shell"
	"github.com/usbarmory/tamago/soc/nxp/usdhc"
)

const (
	expectedBlockSize = 512 // Expected size of MMC block in bytes
	batchSize         = 2048
	osBlock           = 2
)

func init() {
	shell.Add(shell.Cmd{
		Name: "flash",
		Args:    1,
		Pattern: regexp.MustCompile(`^flash (.*)`),
		Syntax:  "<path>",
		Help: "update omniwitness firmware",
		Fn:   flashCmd,
	})
}

func flash(card *usdhc.USDHC, buf []byte, lba int) (err error) {
	blockSize := card.Info().BlockSize
	if blockSize != expectedBlockSize {
		return fmt.Errorf("h/w invariant error - expected MMC blocksize %d, found %d", expectedBlockSize, blockSize)
	}

	// write in chunks to limit DMA requirements
	bytesPerChunk := blockSize * batchSize
	for blocks := 0; len(buf) > 0; {
		var chunk []byte
		if len(buf) >= bytesPerChunk {
			chunk = buf[:bytesPerChunk]
			buf = buf[bytesPerChunk:]
		} else {
			// The final chunk could end with a partial MMC block, so it may need padding with zeroes to make up
			// a whole MMC block size. We'll do this with a separate buffer rather than trying to extend the
			// passed-in buf as doing so will potentially cause a re-alloc & copy which would temporarily use double
			// the amount of RAM.
			roundedUpSize := ((len(buf) / blockSize) + 1) * blockSize
			chunk = make([]byte, roundedUpSize)
			copy(chunk, buf)
			buf = []byte{}
		}
		if err = card.WriteBlocks(lba+blocks, chunk); err != nil {
			return
		}
		blocks += len(chunk) / blockSize

		log.Printf("flashed %d blocks", blocks)
	}

	return
}

func flashCmd(_ *shell.Interface, arg []string) (res string, err error) {
	buf, err := os.ReadFile(arg[0])

	if err != nil {
		return "", fmt.Errorf("could not read file, %v", err)
	}

	card := MMC[0]

	if err = card.Detect(); err != nil {
		return
	}

	log.Printf("flashing %s (%d bytes) @ 0x%x", arg[0], len(buf), osBlock)

	if err = flash(card, buf, osBlock); err != nil {
		return "", fmt.Errorf("flashing error, %v", err)
	}

	return
}
