package render

import (
	"bytes"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
)

var displayP3CICPChunkData = []byte{12, 13, 0, 1}

func writePNG(outputPath string, img image.Image) error {
	return writePNGWithDPI(outputPath, img, defaultOutputDPI)
}

func writePNGWithDPI(outputPath string, img image.Image, dpi int) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := newPNGMetadataWriter(file, normalizeOutputDPI(dpi))
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(writer, img); err != nil {
		return err
	}
	return nil
}

type pngMetadataWriter struct {
	dst      io.Writer
	dpi      int
	prefix   []byte
	inserted bool
}

func newPNGMetadataWriter(dst io.Writer, dpi int) *pngMetadataWriter {
	return &pngMetadataWriter{dst: dst, dpi: dpi, prefix: make([]byte, 0, 33)}
}

func (writer *pngMetadataWriter) Write(data []byte) (int, error) {
	written := len(data)
	for len(data) > 0 {
		if !writer.inserted {
			need := 33 - len(writer.prefix)
			if need > len(data) {
				writer.prefix = append(writer.prefix, data...)
				return written, nil
			}
			writer.prefix = append(writer.prefix, data[:need]...)
			data = data[need:]
			output := pngWithOutputMetadataPrefix(writer.prefix, writer.dpi)
			if _, err := writer.dst.Write(output); err != nil {
				return 0, err
			}
			writer.inserted = true
			continue
		}
		if _, err := writer.dst.Write(data); err != nil {
			return 0, err
		}
		break
	}
	return written, nil
}

func pngWithOutputMetadata(data []byte, dpi int) []byte {
	if len(data) < 33 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return data
	}
	if string(data[12:16]) != "IHDR" {
		return data
	}
	pixelsPerMeter := uint32(math.Round(float64(normalizeOutputDPI(dpi)) / 0.0254))
	chunkData := make([]byte, 9)
	writeUint32BE(chunkData[0:4], pixelsPerMeter)
	writeUint32BE(chunkData[4:8], pixelsPerMeter)
	chunkData[8] = 1
	colorChunk := pngChunk("cICP", displayP3CICPChunkData)
	densityChunk := pngChunk("pHYs", chunkData)
	output := make([]byte, 0, len(data)+len(colorChunk)+len(densityChunk))
	output = append(output, data[:33]...)
	output = append(output, colorChunk...)
	output = append(output, densityChunk...)
	output = append(output, data[33:]...)
	return output
}

func pngWithOutputMetadataPrefix(prefix []byte, dpi int) []byte {
	if len(prefix) != 33 {
		return prefix
	}
	return pngWithOutputMetadata(prefix, dpi)
}

func pngChunk(chunkType string, data []byte) []byte {
	chunk := make([]byte, 8+len(data)+4)
	writeUint32BE(chunk[0:4], uint32(len(data)))
	copy(chunk[4:8], chunkType)
	copy(chunk[8:8+len(data)], data)
	crc := crc32.NewIEEE()
	_, _ = crc.Write(chunk[4 : 8+len(data)])
	writeUint32BE(chunk[8+len(data):], crc.Sum32())
	return chunk
}

func writeUint32BE(data []byte, value uint32) {
	if len(data) < 4 {
		return
	}
	data[0] = byte(value >> 24)
	data[1] = byte(value >> 16)
	data[2] = byte(value >> 8)
	data[3] = byte(value)
}
