package syncer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/media"
	"github.com/dropz/dropz/internal/model"
)

const trimTimeout = 10 * time.Minute

// VideoKeyframes returns the keyframe times of a library clip, read from
// its sample tables, and its duration. The trim UI snaps the in point to
// these so the saved cut is exactly what the timeline showed.
func (c *Coordinator) VideoKeyframes(src string) ([]int64, int64, error) {
	info, err := media.Probe(src)
	if err != nil {
		return nil, 0, err
	}
	v := info.Video()
	if v == nil {
		return nil, 0, errors.New("no video track")
	}
	dur := v.Duration
	if dur == 0 {
		dur = info.Duration
	}
	return v.Keyframes, dur, nil
}

// TrimVideo writes [startMs, endMs] of a library clip as a new file next
// to it, by stream copy: no re-encode, so the cut starts on the keyframe
// at or before startMs (snapped here, whatever the caller sent) and the
// video, audio, and GPMF telemetry tracks are byte-identical. The
// original is never touched. Best effort afterwards: a thumbnail, and a
// preview cut from the source's proxy so the new clip plays at once.
func (c *Coordinator) TrimVideo(src string, startMs, endMs int64) (*model.TrimResult, error) {
	ff := ffmpegPath()
	if ff == "" {
		return nil, errors.New("ffmpeg not found next to the binary or in PATH")
	}
	info, err := media.Probe(src)
	if err != nil {
		return nil, fmt.Errorf("read clip: %w", err)
	}
	video := info.Video()
	if video == nil {
		return nil, errors.New("no video track")
	}
	duration := video.Duration
	if duration == 0 {
		duration = info.Duration
	}
	if startMs < 0 {
		startMs = 0
	}
	if endMs <= 0 || endMs > duration {
		endMs = duration
	}
	if endMs-startMs < 500 {
		return nil, errors.New("trim shorter than half a second")
	}
	startMs = snapToKeyframe(video.Keyframes, startMs)
	if startMs == 0 && endMs >= duration {
		return nil, errors.New("trim covers the whole clip")
	}

	// ffmpeg's stream index is the trak order. The timecode track (tmcd)
	// has no writable codec and would fail the whole mux; everything else
	// the camera wrote comes along.
	args := []string{"-y", "-v", "error",
		"-ss", msArg(startMs), "-to", msArg(endMs), "-i", src}
	for _, t := range info.Tracks {
		if t.Codec == "tmcd" {
			continue
		}
		args = append(args, "-map", fmt.Sprintf("0:%d", t.Index))
	}

	dir := filepath.Dir(src)
	base := filepath.Base(src)
	outName := trimName(dir, base, startMs, endMs)
	out := filepath.Join(dir, outName)
	tmp := out + ".partial"
	args = append(args, "-c", "copy", "-movflags", "+faststart", "-f", "mp4", tmp)

	ctx, cancel := context.WithTimeout(c.ctx, trimTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ff, args...)
	if outp, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(string(outp)))
	}
	if err := os.Rename(tmp, out); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	fi, err := os.Stat(out)
	if err != nil {
		return nil, err
	}
	c.log.Info("Clip trimmed", "src", base, "out", outName, "start_ms", startMs, "end_ms", endMs,
		"mb", fmt.Sprintf("%.1f", float64(fi.Size())/1e6))

	result := &model.TrimResult{
		OutputPath: out,
		Name:       outName,
		SizeBytes:  fi.Size(),
		StartMs:    startMs,
		EndMs:      endMs,
	}

	thumb := ThumbnailPath(dir, outName)
	if err := os.MkdirAll(filepath.Dir(thumb), 0755); err == nil {
		tcmd := exec.CommandContext(ctx, ff, "-y", "-v", "error", "-i", out,
			"-frames:v", "1", "-vf", "scale=480:-2", "-q:v", "4", thumb)
		if outp, err := tcmd.CombinedOutput(); err != nil {
			c.log.Debug("Trim thumbnail failed", "out", outName, "err", err, "ffmpeg", strings.TrimSpace(string(outp)))
		} else {
			result.ThumbnailPath = thumb
		}
	}

	// The proxy is 480p VP9 already; re-encoding a slice of it takes a
	// few seconds and keeps the new clip playable in-app without
	// decoding the full-res original.
	if srcPreview := PreviewPath(dir, base); fileExists(srcPreview) {
		preview := PreviewPath(dir, outName)
		ptmp := preview + ".partial"
		pcmd := exec.CommandContext(ctx, ff, "-y", "-v", "error",
			"-ss", msArg(startMs), "-to", msArg(endMs), "-i", srcPreview,
			"-c:v", "libvpx-vp9", "-row-mt", "1", "-deadline", "realtime", "-cpu-used", "7",
			"-crf", "34", "-b:v", "0", "-c:a", "libopus", "-b:a", "64k", "-f", "webm", ptmp)
		if outp, err := pcmd.CombinedOutput(); err != nil {
			_ = os.Remove(ptmp)
			c.log.Debug("Trim preview failed", "out", outName, "err", err, "ffmpeg", strings.TrimSpace(string(outp)))
		} else if err := os.Rename(ptmp, preview); err == nil {
			result.PreviewPath = preview
		}
	}
	return result, nil
}

// snapToKeyframe returns the last keyframe at or before ms; a file
// without a sync table is all keyframes, so ms stands.
func snapToKeyframe(keyframes []int64, ms int64) int64 {
	if len(keyframes) == 0 {
		return ms
	}
	best := keyframes[0]
	for _, k := range keyframes {
		if k > ms {
			break
		}
		best = k
	}
	return best
}

func msArg(ms int64) string {
	return fmt.Sprintf("%d.%03d", ms/1000, ms%1000)
}

// trimName is "<base>_trim-<mmss>-<mmss><ext>", with a counter when a
// cut with the same bounds already exists.
func trimName(dir, base string, startMs, endMs int64) string {
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stamp := func(ms int64) string {
		s := ms / 1000
		return fmt.Sprintf("%02d%02d", s/60, s%60)
	}
	name := fmt.Sprintf("%s_trim-%s-%s", stem, stamp(startMs), stamp(endMs))
	candidate := name + ext
	for n := 2; fileExists(filepath.Join(dir, candidate)); n++ {
		candidate = fmt.Sprintf("%s-%d%s", name, n, ext)
	}
	return candidate
}
