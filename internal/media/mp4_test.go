package media

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// Box builders for a synthetic file laid out like a GoPro clip: ftyp,
// a large mdat, then moov at the end.

func atom(typ string, parts ...[]byte) []byte {
	body := bytes.Join(parts, nil)
	out := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(out[:4], uint32(8+len(body)))
	copy(out[4:8], typ)
	return append(out, body...)
}

func u32(v ...uint32) []byte {
	out := make([]byte, 4*len(v))
	for i, x := range v {
		binary.BigEndian.PutUint32(out[i*4:], x)
	}
	return out
}

func fullBox(typ string, version byte, parts ...[]byte) []byte {
	return atom(typ, append([][]byte{{version, 0, 0, 0}}, parts...)...)
}

// mdhd v0: creation, modification, timescale, duration, language+quality
func mdhd(timescale, duration uint32) []byte {
	return fullBox("mdhd", 0, u32(0, 0, timescale, duration), []byte{0, 0, 0, 0})
}

func hdlr(handler string) []byte {
	return fullBox("hdlr", 0, u32(0), []byte(handler), u32(0, 0, 0), []byte{0})
}

func stsd(codec string) []byte {
	entry := atom(codec, make([]byte, 16))
	return fullBox("stsd", 0, u32(1), entry)
}

func table(typ string, entries ...uint32) []byte {
	return fullBox(typ, 0, u32(uint32(len(entries)/2)), u32(entries...))
}

func trak(handler, codec string, timescale, duration uint32, stbl ...[]byte) []byte {
	return atom("trak",
		atom("mdia",
			mdhd(timescale, duration),
			hdlr(handler),
			atom("minf", atom("stbl", append([][]byte{stsd(codec)}, stbl...)...)),
		),
	)
}

func writeSample(t *testing.T) string {
	t.Helper()
	// Video: 24000 timescale, 10 samples of 1001, sync samples 1, 4, 7,
	// ctts offset 2002 on every sample (B-frame style reorder delay).
	video := trak("vide", "hvc1", 24000, 10010,
		table("stts", 10, 1001),
		fullBox("stss", 0, u32(3), u32(1, 4, 7)),
		table("ctts", 10, 2002),
	)
	audio := trak("soun", "mp4a", 48000, 20000, table("stts", 20, 1000))
	tmcd := trak("tmcd", "tmcd", 24000, 10010, table("stts", 1, 10010))
	gpmd := trak("meta", "gpmd", 1000, 417, table("stts", 1, 417))
	// mvhd v0: creation, modification, timescale, duration
	mvhd := fullBox("mvhd", 0, u32(0, 0, 1000, 417))
	moov := atom("moov", mvhd, video, audio, tmcd, gpmd)

	// Large mdat via the 64-bit size form, padded with zeros
	mdatBody := make([]byte, 4096)
	mdat := make([]byte, 16)
	binary.BigEndian.PutUint32(mdat[:4], 1)
	copy(mdat[4:8], "mdat")
	binary.BigEndian.PutUint64(mdat[8:16], uint64(16+len(mdatBody)))
	mdat = append(mdat, mdatBody...)

	file := bytes.Join([][]byte{atom("ftyp", []byte("mp42"), u32(0)), mdat, moov}, nil)
	path := filepath.Join(t.TempDir(), "GX010001.MP4")
	if err := os.WriteFile(path, file, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProbeKeyframesAndTracks(t *testing.T) {
	info, err := Probe(writeSample(t))
	if err != nil {
		t.Fatal(err)
	}
	if info.Duration != 417 {
		t.Errorf("duration = %d ms, want 417", info.Duration)
	}
	if len(info.Tracks) != 4 {
		t.Fatalf("tracks = %d, want 4", len(info.Tracks))
	}
	wantCodec := []string{"hvc1", "mp4a", "tmcd", "gpmd"}
	wantHandler := []string{"vide", "soun", "tmcd", "meta"}
	for i, tr := range info.Tracks {
		if tr.Index != i || tr.Codec != wantCodec[i] || tr.Handler != wantHandler[i] {
			t.Errorf("track %d = %+v", i, tr)
		}
	}
	v := info.Video()
	if v == nil {
		t.Fatal("no video track")
	}
	// Sample n starts at (n-1)*1001 dts; +2002 ctts; in ms at 24000 Hz:
	// sample 1: 2002/24 = 83, sample 4: (3003+2002)/24 = 208, sample 7: 333
	want := []int64{83, 208, 333}
	if len(v.Keyframes) != len(want) {
		t.Fatalf("keyframes = %v, want %v", v.Keyframes, want)
	}
	for i := range want {
		if v.Keyframes[i] != want[i] {
			t.Errorf("keyframe %d = %d, want %d", i, v.Keyframes[i], want[i])
		}
	}
	if v.Duration != 417 {
		t.Errorf("video duration = %d, want 417", v.Duration)
	}
}

func TestProbeRejectsNonMP4(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.MP4")
	if err := os.WriteFile(path, []byte("not an mp4 at all, just text"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Probe(path); err == nil {
		t.Error("garbage accepted")
	}
}
