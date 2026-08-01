package album

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

// ExifData missing values are zero; TakenAt.IsZero() means "fall back to upload time".
type ExifData struct {
	TakenAt   time.Time
	Latitude  *float64
	Longitude *float64
}

type ExifExtractor interface {
	Extract(ctx context.Context, src []byte, mime string) (ExifData, error)
}

type stdlibExtractor struct{}

func NewExifExtractor() ExifExtractor { return &stdlibExtractor{} }

func (e *stdlibExtractor) Extract(_ context.Context, src []byte, mime string) (ExifData, error) {
	if mime != "image/jpeg" {
		return ExifData{}, nil
	}
	tiff, err := findExifTiff(src)
	if err != nil || len(tiff) == 0 {
		return ExifData{}, nil
	}
	return parseTiff(tiff), nil
}

func findExifTiff(src []byte) ([]byte, error) {
	if len(src) < 4 || src[0] != 0xFF || src[1] != 0xD8 {
		return nil, errors.New("not jpeg")
	}
	i := 2
	for i+4 <= len(src) {
		if src[i] != 0xFF {
			return nil, nil
		}
		marker := src[i+1]
		i += 2
		// SOI/EOI carry no segment length; SOS begins compressed image data,
		// so anything past it is not a metadata segment.
		if marker == 0xD8 || marker == 0xD9 {
			continue
		}
		if marker == 0xDA {
			return nil, nil
		}
		if i+2 > len(src) {
			return nil, io.ErrUnexpectedEOF
		}
		segLen := int(binary.BigEndian.Uint16(src[i : i+2]))
		if segLen < 2 || i+segLen > len(src) {
			return nil, io.ErrUnexpectedEOF
		}
		if marker == 0xE1 && segLen >= 8 && bytes.HasPrefix(src[i+2:i+segLen], []byte("Exif\x00\x00")) {
			return src[i+8 : i+segLen], nil
		}
		i += segLen
	}
	return nil, nil
}

func parseTiff(tiff []byte) ExifData {
	if len(tiff) < 8 {
		return ExifData{}
	}
	var bo binary.ByteOrder
	switch string(tiff[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return ExifData{}
	}
	if bo.Uint16(tiff[2:4]) != 42 {
		return ExifData{}
	}
	ifd0Off := bo.Uint32(tiff[4:8])
	if ifd0Off < 8 || int(ifd0Off) >= len(tiff) {
		return ExifData{}
	}

	out := ExifData{}
	// Walk IFD0 to find ExifIFD (0x8769) and GPSIFD (0x8825) offsets.
	exifIFDOff, gpsIFDOff := walkForSubIFDs(tiff, bo, int(ifd0Off))
	if exifIFDOff > 0 {
		if t := readDateTimeOriginal(tiff, bo, exifIFDOff); !t.IsZero() {
			out.TakenAt = t
		}
	}
	if gpsIFDOff > 0 {
		lat, lng := readGPS(tiff, bo, gpsIFDOff)
		out.Latitude = lat
		out.Longitude = lng
	}
	return out
}

func walkForSubIFDs(tiff []byte, bo binary.ByteOrder, off int) (exifIFD, gpsIFD int) {
	if off+2 > len(tiff) {
		return
	}
	n := int(bo.Uint16(tiff[off : off+2]))
	base := off + 2
	for i := 0; i < n; i++ {
		p := base + i*12
		if p+12 > len(tiff) {
			return
		}
		tag := bo.Uint16(tiff[p : p+2])
		typ := bo.Uint16(tiff[p+2 : p+4])
		val := bo.Uint32(tiff[p+8 : p+12])
		if typ == 4 { // LONG
			switch tag {
			case 0x8769:
				exifIFD = int(val)
			case 0x8825:
				gpsIFD = int(val)
			}
		}
	}
	return
}

func readDateTimeOriginal(tiff []byte, bo binary.ByteOrder, off int) time.Time {
	if off+2 > len(tiff) {
		return time.Time{}
	}
	n := int(bo.Uint16(tiff[off : off+2]))
	base := off + 2
	for i := 0; i < n; i++ {
		p := base + i*12
		if p+12 > len(tiff) {
			return time.Time{}
		}
		tag := bo.Uint16(tiff[p : p+2])
		if tag != 0x9003 { // DateTimeOriginal
			continue
		}
		typ := bo.Uint16(tiff[p+2 : p+4])
		if typ != 2 { // must be ASCII
			return time.Time{}
		}
		count := bo.Uint32(tiff[p+4 : p+8])
		if count == 0 || count > 32 {
			return time.Time{}
		}
		valOff := int(bo.Uint32(tiff[p+8 : p+12]))
		if valOff+int(count) > len(tiff) {
			return time.Time{}
		}
		s := string(bytes.TrimRight(tiff[valOff:valOff+int(count)], "\x00 "))
		if t, err := time.Parse("2006:01:02 15:04:05", s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func readGPS(tiff []byte, bo binary.ByteOrder, off int) (*float64, *float64) {
	if off+2 > len(tiff) {
		return nil, nil
	}
	n := int(bo.Uint16(tiff[off : off+2]))
	base := off + 2
	var latRef, lngRef byte
	var latVals, lngVals []float64
	var latOK, lngOK bool
	for i := 0; i < n; i++ {
		p := base + i*12
		if p+12 > len(tiff) {
			return nil, nil
		}
		tag := bo.Uint16(tiff[p : p+2])
		typ := bo.Uint16(tiff[p+2 : p+4])
		count := bo.Uint32(tiff[p+4 : p+8])
		valOff := int(bo.Uint32(tiff[p+8 : p+12]))
		switch tag {
		case 0x0001: // GPSLatitudeRef
			if typ == 2 && count > 0 {
				latRef = firstByteInline(tiff, p)
			}
		case 0x0002: // GPSLatitude — 3 RATIONALs (deg/min/sec)
			if typ == 5 && count == 3 {
				latVals, latOK = readRationals(tiff, bo, valOff, 3)
			}
		case 0x0003: // GPSLongitudeRef
			if typ == 2 && count > 0 {
				lngRef = firstByteInline(tiff, p)
			}
		case 0x0004: // GPSLongitude
			if typ == 5 && count == 3 {
				lngVals, lngOK = readRationals(tiff, bo, valOff, 3)
			}
		}
	}
	if !latOK || !lngOK || !validGPSRef(latRef, "NS") || !validGPSRef(lngRef, "EW") {
		return nil, nil
	}
	lat, ok := dmsToDeg(latVals, latRef == 'S', 90)
	if !ok {
		return nil, nil
	}
	lng, ok := dmsToDeg(lngVals, lngRef == 'W', 180)
	if !ok {
		return nil, nil
	}
	// Null Island: cameras write (0, 0) when GPS never locked. Treat as absent.
	if lat == 0 && lng == 0 {
		return nil, nil
	}
	return &lat, &lng
}

func validGPSRef(b byte, allowed string) bool {
	for i := 0; i < len(allowed); i++ {
		if b == allowed[i] {
			return true
		}
	}
	return false
}

// firstByteInline reads the first byte of an ASCII/BYTE tag stored inline in
// the 4-byte value slot (short strings fit without an out-of-band pointer).
func firstByteInline(tiff []byte, tagOff int) byte {
	return tiff[tagOff+8]
}

// readRationals returns (values, true) on success, or (nil, false) if any
// denominator is zero or the offset is out of bounds.
func readRationals(tiff []byte, bo binary.ByteOrder, off, count int) ([]float64, bool) {
	if off < 0 || off+count*8 > len(tiff) {
		return nil, false
	}
	out := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		p := off + i*8
		num := bo.Uint32(tiff[p : p+4])
		den := bo.Uint32(tiff[p+4 : p+8])
		if den == 0 {
			return nil, false
		}
		out = append(out, float64(num)/float64(den))
	}
	return out, true
}

// dmsToDeg converts (deg, min, sec) to signed decimal degrees, rejecting
// out-of-range components (min/sec must be < 60, final magnitude ≤ maxAbs).
func dmsToDeg(dms []float64, negative bool, maxAbs float64) (float64, bool) {
	if len(dms) != 3 {
		return 0, false
	}
	if dms[0] < 0 || dms[1] < 0 || dms[2] < 0 {
		return 0, false
	}
	if dms[1] >= 60 || dms[2] >= 60 {
		return 0, false
	}
	deg := dms[0] + dms[1]/60 + dms[2]/3600
	if deg > maxAbs {
		return 0, false
	}
	if negative {
		deg = -deg
	}
	return deg, true
}
