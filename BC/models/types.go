package models

import "sync"

// AudioTrack represents an audio stream track
type AudioTrack struct {
	Index int    `json:"index"`
	Lang  string `json:"lang"`
	Title string `json:"title"`
}

// Chapters represents chapter markers with their timestamps
type Chapters map[string]float64

// ScanResult contains the result of scanning video files
type ScanResult struct {
	Chapters    Chapters     `json:"chapters"`
	AudioTracks []AudioTrack `json:"audioTracks"`
	FirstFile   string       `json:"firstFile"`
}

// SkipRange defines a range to skip in the video
type SkipRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// TrimOptions contains options for trimming operations
type TrimOptions struct {
	SkipRanges []SkipRange `json:"skipRanges"`
	Parts      int         `json:"parts"`
	AudioIndex int         `json:"audioIndex"` // Default audio track (not used for removal, just for reference)
}

// Progress tracks the progress of video processing
type Progress struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Percent   float64 `json:"percent"`
	Status    string  `json:"status"`
	Done      bool    `json:"done"`
	mu        sync.Mutex
}

// ProgressState is the global progress state
var ProgressState = &Progress{Status: "idle"}

// Update updates the progress state safely
func (p *Progress) Update(fn func(*Progress)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn(p)
}

// Get gets the current progress state safely
func (p *Progress) Get(fn func(*Progress)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn(p)
}

// MetaChapter represents a single chapter parsed from ffmetadata
type MetaChapter struct {
	Start int64
	End   int64
	Title string
}

// MetaFile contains chapters and the timebase numerator/denominator
type MetaFile struct {
	TimebaseNum int64
	TimebaseDen int64
	Chapters    []MetaChapter
}
