package helpers

import (
	"io"
)

// Static zero byte buffer that are used for filling free space in files.
var zeros = [100]byte{}

// /dev/null
var dump = [100]byte{}

// Effectively writes continuous range of zero-filled bytes into a writer.
func WriteZeros(size int, writer io.Writer) (int, error) {
	total := 0
	for total < size {
		chunk := size - total
		if chunk > len(zeros) {
			chunk = len(zeros)
		}
		written, err := writer.Write(zeros[:chunk])
		total += written
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Skips some number of bytes in io.Reader.
func SkipReading(size int, reader io.Reader) (int, error) {
	if size == 0 {
		return 0, nil
	}
	skipped := 0
	for skipped < size {
		chunk := size - skipped
		if chunk > len(dump) {
			chunk = len(dump)
		}
		read, err := reader.Read(dump[:chunk])
		skipped += read
		if err != nil {
			return skipped, err
		}
	}
	return skipped, nil
}
