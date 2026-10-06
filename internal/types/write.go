package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"unicode/utf8"
)

// BodyImageLimit is exclusive: uploadimg requires jpg/png smaller than 1 MB.
const BodyImageLimit = 1 << 20

type BodyImageResponse struct {
	URL string `json:"url"`
}

// ValidateBodyImage inspects the original bytes; it never encodes/resizes them.
func ValidateBodyImage(data []byte) error {
	if len(data) == 0 || len(data) >= BodyImageLimit {
		return fmt.Errorf("body image must be nonempty and smaller than 1 MB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return fmt.Errorf("body image must be jpg or png")
	}
	// Bound decoding memory and reject header-only/truncated files without reencoding.
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 64000000 {
		return fmt.Errorf("body image exceeds safe decoded pixel limit")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("body image must contain complete decodable pixels")
	}
	return nil
}

// Raw JSON and pointer index distinguish absent fields from explicit empty/zero values.
// Articles is a single complete article, matching WeChat's update endpoint.
type DraftUpdateRequest struct {
	MediaID  string          `json:"media_id"`
	Index    *int            `json:"index"`
	Articles json.RawMessage `json:"articles"`
}

func (r DraftUpdateRequest) Validate() (Article, error) {
	var a Article
	if strings.TrimSpace(r.MediaID) == "" || r.Index == nil || *r.Index < 0 {
		return a, fmt.Errorf("media_id and explicit nonnegative index are required")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(r.Articles, &fields); err != nil || fields == nil {
		return a, fmt.Errorf("articles must be a complete article object")
	}
	for _, name := range []string{"title", "author", "digest", "content", "content_source_url", "thumb_media_id", "show_cover_pic", "need_open_comment", "only_fans_can_comment"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return a, fmt.Errorf("article requires explicit field %s", name)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(r.Articles))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&a); err != nil {
		return a, fmt.Errorf("invalid article fields")
	}
	if strings.TrimSpace(a.Title) == "" || strings.TrimSpace(a.Content) == "" || strings.TrimSpace(a.ThumbMediaID) == "" {
		return a, fmt.Errorf("title, content and thumb_media_id must be nonempty")
	}
	if utf8.RuneCountInString(a.Title) > 64 {
		return a, fmt.Errorf("title must not exceed 64 characters")
	}
	if utf8.RuneCountInString(a.Content) >= 20000 || len(a.Content) >= 1<<20 {
		return a, fmt.Errorf("content must be smaller than 20000 characters and 1 MiB")
	}
	if a.ShowCoverPic > 1 || a.NeedOpenComment > 1 || a.OnlyFansCanComment > 1 {
		return a, fmt.Errorf("article flags must be 0 or 1")
	}
	return a, nil
}
