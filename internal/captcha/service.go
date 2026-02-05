package captcha

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	mathrand "math/rand"
	"sync"
	"time"
)

type Challenge struct {
	ID                    string
	Type                  string
	BackgroundImage       string
	TemplateImage         string
	BackgroundImageHeight int
	Data                  map[string]any
}

type entry struct {
	TargetX   int
	ExpiresAt time.Time
}

type tokenEntry struct {
	ExpiresAt time.Time
}

type Service struct {
	mu      sync.Mutex
	entries map[string]entry
	tokens  map[string]tokenEntry
	TTL     time.Duration
}

func New(ttl time.Duration) *Service {
	return &Service{
		entries: make(map[string]entry),
		tokens:  make(map[string]tokenEntry),
		TTL:     ttl,
	}
}

func (s *Service) Generate() Challenge {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()

	id := randomID()
	bg, piece, targetX := generateSliderImages()
	exp := time.Now().Add(s.TTL)
	s.entries[id] = entry{TargetX: targetX, ExpiresAt: exp}

	return Challenge{
		ID:                    id,
		Type:                  "SLIDER",
		BackgroundImage:       bg,
		TemplateImage:         piece,
		BackgroundImageHeight: 180,
		Data:                  map[string]any{},
	}
}

func (s *Service) VerifyTrack(id string, trackData []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()

	ent, ok := s.entries[id]
	if !ok || time.Now().After(ent.ExpiresAt) {
		delete(s.entries, id)
		return false
	}

	moveX, ok := extractMoveX(trackData)
	if !ok {
		return false
	}

	if math.Abs(moveX-float64(ent.TargetX)) > 6 {
		return false
	}

	delete(s.entries, id)
	s.tokens[id] = tokenEntry{ExpiresAt: time.Now().Add(s.TTL)}
	return true
}

func (s *Service) ConsumeToken(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()

	entry, ok := s.tokens[id]
	if !ok || time.Now().After(entry.ExpiresAt) {
		delete(s.tokens, id)
		return false
	}
	delete(s.tokens, id)
	return true
}

func (s *Service) cleanupLocked() {
	now := time.Now()
	for id, e := range s.entries {
		if now.After(e.ExpiresAt) {
			delete(s.entries, id)
		}
	}
	for id, t := range s.tokens {
		if now.After(t.ExpiresAt) {
			delete(s.tokens, id)
		}
	}
}

func extractMoveX(trackData []byte) (float64, bool) {
	if len(trackData) == 0 {
		return 0, false
	}

	var payload struct {
		TrackList []struct {
			X float64 `json:"x"`
		} `json:"trackList"`
		MoveX float64 `json:"moveX"`
	}
	if err := json.Unmarshal(trackData, &payload); err == nil {
		if len(payload.TrackList) > 0 {
			maxX := payload.TrackList[0].X
			for _, p := range payload.TrackList {
				if p.X > maxX {
					maxX = p.X
				}
			}
			return maxX, true
		}
		if payload.MoveX != 0 {
			return payload.MoveX, true
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(trackData, &raw); err != nil {
		return 0, false
	}
	if list, ok := raw["trackList"].([]any); ok {
		maxX := float64(0)
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				if x, ok := m["x"].(float64); ok && x > maxX {
					maxX = x
				}
			}
		}
		if maxX > 0 {
			return maxX, true
		}
	}
	if v, ok := raw["moveX"].(float64); ok {
		return v, true
	}
	return 0, false
}

func generateSliderImages() (string, string, int) {
	const (
		width     = 300
		height    = 180
		pieceSize = 50
	)

	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	bg := image.NewRGBA(image.Rect(0, 0, width, height))
	base := color.RGBA{uint8(80 + rng.Intn(120)), uint8(80 + rng.Intn(120)), uint8(80 + rng.Intn(120)), 255}
	draw.Draw(bg, bg.Bounds(), &image.Uniform{base}, image.Point{}, draw.Src)

	for i := 0; i < 32; i++ {
		x0 := rng.Intn(width)
		y0 := rng.Intn(height)
		x1 := x0 + 10 + rng.Intn(40)
		y1 := y0 + 10 + rng.Intn(40)
		col := color.RGBA{uint8(rng.Intn(255)), uint8(rng.Intn(255)), uint8(rng.Intn(255)), uint8(80 + rng.Intn(80))}
		draw.Draw(bg, image.Rect(x0, y0, minInt(x1, width), minInt(y1, height)), &image.Uniform{col}, image.Point{}, draw.Over)
	}

	maxX := width - pieceSize - 10
	if maxX > 242 {
		maxX = 242
	}
	targetX := 10 + rng.Intn(maxX-9)
	targetY := 10 + rng.Intn(height-pieceSize-20)

	piece := image.NewRGBA(image.Rect(0, 0, pieceSize, pieceSize))
	draw.Draw(piece, piece.Bounds(), bg, image.Point{X: targetX, Y: targetY}, draw.Src)

	gap := color.RGBA{255, 255, 255, 140}
	draw.Draw(bg, image.Rect(targetX, targetY, targetX+pieceSize, targetY+pieceSize), &image.Uniform{gap}, image.Point{}, draw.Over)

	bgData := encodePNGDataURL(bg)
	pieceData := encodePNGDataURL(piece)
	return bgData, pieceData, targetX
}

func encodePNGDataURL(img image.Image) string {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
