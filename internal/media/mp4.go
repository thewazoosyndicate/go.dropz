// Package media reads what the trim UI needs straight from an MP4's
// sample tables, so keyframe positions come from the file's own index
// in milliseconds without decoding a frame. GoPro files carry the
// moov atom after a multi-gigabyte mdat, so boxes are walked by size.
package media

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// maxMoov bounds the atom read into memory; a real GoPro moov is a few
// hundred KB per hour of footage.
const maxMoov = 64 << 20

// Track is one trak of the file, in file order, which is also the
// stream index ffmpeg assigns.
type Track struct {
	Index     int
	Handler   string // vide, soun, meta, ...
	Codec     string // first stsd entry: hvc1, avc1, mp4a, gpmd, tmcd
	Timescale uint32
	Duration  int64   // ms
	Keyframes []int64 // ms, ascending; nil when every sample is a sync sample
}

// Info is the file-level result of Probe.
type Info struct {
	Duration int64 // ms, from mvhd
	Tracks   []Track
}

// Video returns the first video track, or nil.
func (i *Info) Video() *Track {
	for idx := range i.Tracks {
		if i.Tracks[idx].Handler == "vide" {
			return &i.Tracks[idx]
		}
	}
	return nil
}

// Probe parses the moov atom of an MP4 file.
func Probe(path string) (*Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	moov, err := findMoov(f)
	if err != nil {
		return nil, err
	}
	return parseMoov(moov)
}

// findMoov walks the top-level boxes and returns the moov body.
func findMoov(r io.ReadSeeker) ([]byte, error) {
	var hdr [16]byte
	for {
		if _, err := io.ReadFull(r, hdr[:8]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, errors.New("no moov atom found")
			}
			return nil, err
		}
		size := uint64(binary.BigEndian.Uint32(hdr[:4]))
		typ := string(hdr[4:8])
		headerLen := uint64(8)
		switch size {
		case 1:
			if _, err := io.ReadFull(r, hdr[8:16]); err != nil {
				return nil, err
			}
			size = binary.BigEndian.Uint64(hdr[8:16])
			headerLen = 16
		case 0:
			// Box runs to end of file; only meaningful for a trailing mdat
			if typ != "moov" {
				return nil, errors.New("no moov atom found")
			}
			body, err := io.ReadAll(io.LimitReader(r, maxMoov))
			return body, err
		}
		if size < headerLen {
			return nil, fmt.Errorf("corrupt box %q: size %d", typ, size)
		}
		body := size - headerLen
		if typ == "moov" {
			if body > maxMoov {
				return nil, fmt.Errorf("moov atom too large: %d bytes", body)
			}
			buf := make([]byte, body)
			if _, err := io.ReadFull(r, buf); err != nil {
				return nil, err
			}
			return buf, nil
		}
		if _, err := r.Seek(int64(body), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
}

// box is a parsed child within a buffer.
type box struct {
	typ  string
	body []byte
}

// children splits a container body into its boxes.
func children(buf []byte) ([]box, error) {
	var out []box
	for len(buf) >= 8 {
		size := uint64(binary.BigEndian.Uint32(buf[:4]))
		typ := string(buf[4:8])
		headerLen := uint64(8)
		if size == 1 {
			if len(buf) < 16 {
				return nil, fmt.Errorf("truncated box %q", typ)
			}
			size = binary.BigEndian.Uint64(buf[8:16])
			headerLen = 16
		} else if size == 0 {
			size = uint64(len(buf))
		}
		if size < headerLen || size > uint64(len(buf)) {
			return nil, fmt.Errorf("corrupt box %q: size %d of %d", typ, size, len(buf))
		}
		out = append(out, box{typ: typ, body: buf[headerLen:size]})
		buf = buf[size:]
	}
	return out, nil
}

func find(boxes []box, typ string) *box {
	for i := range boxes {
		if boxes[i].typ == typ {
			return &boxes[i]
		}
	}
	return nil
}

func parseMoov(buf []byte) (*Info, error) {
	boxes, err := children(buf)
	if err != nil {
		return nil, err
	}
	info := &Info{}
	if mvhd := find(boxes, "mvhd"); mvhd != nil {
		ts, dur, err := fullHeaderTimes(mvhd.body)
		if err != nil {
			return nil, fmt.Errorf("mvhd: %w", err)
		}
		info.Duration = toMs(dur, ts)
	}
	for _, b := range boxes {
		if b.typ != "trak" {
			continue
		}
		t, err := parseTrak(b.body)
		if err != nil {
			return nil, fmt.Errorf("trak %d: %w", len(info.Tracks), err)
		}
		t.Index = len(info.Tracks)
		info.Tracks = append(info.Tracks, t)
	}
	if len(info.Tracks) == 0 {
		return nil, errors.New("no tracks in moov")
	}
	return info, nil
}

// fullHeaderTimes reads timescale and duration from mvhd or mdhd, whose
// layouts share the version-dependent field sizes.
func fullHeaderTimes(body []byte) (timescale uint32, duration uint64, err error) {
	if len(body) < 4 {
		return 0, 0, errors.New("truncated")
	}
	switch body[0] {
	case 0:
		if len(body) < 20 {
			return 0, 0, errors.New("truncated v0")
		}
		return binary.BigEndian.Uint32(body[12:16]), uint64(binary.BigEndian.Uint32(body[16:20])), nil
	case 1:
		if len(body) < 32 {
			return 0, 0, errors.New("truncated v1")
		}
		return binary.BigEndian.Uint32(body[20:24]), binary.BigEndian.Uint64(body[24:32]), nil
	default:
		return 0, 0, fmt.Errorf("unknown version %d", body[0])
	}
}

func toMs(units uint64, timescale uint32) int64 {
	if timescale == 0 {
		return 0
	}
	return int64(units * 1000 / uint64(timescale))
}

func parseTrak(buf []byte) (Track, error) {
	var t Track
	boxes, err := children(buf)
	if err != nil {
		return t, err
	}
	mdia := find(boxes, "mdia")
	if mdia == nil {
		return t, errors.New("no mdia")
	}
	mboxes, err := children(mdia.body)
	if err != nil {
		return t, err
	}
	if mdhd := find(mboxes, "mdhd"); mdhd != nil {
		ts, dur, err := fullHeaderTimes(mdhd.body)
		if err != nil {
			return t, fmt.Errorf("mdhd: %w", err)
		}
		t.Timescale = ts
		t.Duration = toMs(dur, ts)
	}
	if hdlr := find(mboxes, "hdlr"); hdlr != nil && len(hdlr.body) >= 12 {
		t.Handler = string(hdlr.body[8:12])
	}
	minf := find(mboxes, "minf")
	if minf == nil {
		return t, nil
	}
	iboxes, err := children(minf.body)
	if err != nil {
		return t, err
	}
	stbl := find(iboxes, "stbl")
	if stbl == nil {
		return t, nil
	}
	sboxes, err := children(stbl.body)
	if err != nil {
		return t, err
	}
	if stsd := find(sboxes, "stsd"); stsd != nil && len(stsd.body) >= 16 {
		t.Codec = string(stsd.body[12:16])
	}
	if t.Handler != "vide" {
		return t, nil
	}
	stss := find(sboxes, "stss")
	if stss == nil {
		return t, nil // every sample is a sync sample
	}
	stts := find(sboxes, "stts")
	if stts == nil {
		return t, errors.New("video track without stts")
	}
	var ctts *box
	if c := find(sboxes, "ctts"); c != nil {
		ctts = c
	}
	keys, err := keyframeTimes(stts.body, stss.body, cttsBody(ctts), t.Timescale)
	if err != nil {
		return t, err
	}
	t.Keyframes = keys
	return t, nil
}

func cttsBody(b *box) []byte {
	if b == nil {
		return nil
	}
	return b.body
}

// runs is a cursor over an stts/ctts-style (count, value) table.
type runs struct {
	data    []byte
	entries uint32
	i       uint32 // current entry
	left    uint32 // samples left in the current entry
	value   int64
	signed  bool
}

func newRuns(body []byte, signed bool) (*runs, error) {
	if len(body) < 8 {
		return nil, errors.New("truncated table")
	}
	n := binary.BigEndian.Uint32(body[4:8])
	if uint64(len(body)) < 8+uint64(n)*8 {
		return nil, errors.New("table shorter than its count")
	}
	r := &runs{data: body[8:], entries: n, signed: signed}
	r.load()
	return r, nil
}

func (r *runs) load() {
	if r.i >= r.entries {
		r.left = 0
		return
	}
	off := r.i * 8
	r.left = binary.BigEndian.Uint32(r.data[off : off+4])
	v := binary.BigEndian.Uint32(r.data[off+4 : off+8])
	if r.signed {
		r.value = int64(int32(v))
	} else {
		r.value = int64(v)
	}
}

// next returns the value for the next sample; ok is false past the end.
func (r *runs) next() (int64, bool) {
	for r.left == 0 {
		if r.i >= r.entries {
			return 0, false
		}
		r.i++
		r.load()
		if r.i >= r.entries && r.left == 0 {
			return 0, false
		}
	}
	r.left--
	return r.value, true
}

// keyframeTimes walks the samples once, summing stts deltas into dts and
// adding the ctts offset, and emits the presentation time of each sync
// sample listed in stss.
func keyframeTimes(stts, stss, ctts []byte, timescale uint32) ([]int64, error) {
	if len(stss) < 8 {
		return nil, errors.New("truncated stss")
	}
	n := binary.BigEndian.Uint32(stss[4:8])
	if uint64(len(stss)) < 8+uint64(n)*4 {
		return nil, errors.New("stss shorter than its count")
	}
	deltas, err := newRuns(stts, false)
	if err != nil {
		return nil, fmt.Errorf("stts: %w", err)
	}
	var offsets *runs
	if ctts != nil {
		if offsets, err = newRuns(ctts, true); err != nil {
			return nil, fmt.Errorf("ctts: %w", err)
		}
	}
	out := make([]int64, 0, n)
	var dts int64
	sample := uint32(1)
	for k := uint32(0); k < n; k++ {
		want := binary.BigEndian.Uint32(stss[8+k*4 : 12+k*4])
		for sample <= want {
			delta, ok := deltas.next()
			if !ok {
				return nil, fmt.Errorf("stss sample %d beyond stts", want)
			}
			var off int64
			if offsets != nil {
				off, _ = offsets.next()
			}
			if sample == want {
				out = append(out, toMs(uint64(dts+off), timescale))
			}
			dts += delta
			sample++
		}
	}
	return out, nil
}
