package player

import (
	"bytes"
	"encoding/base32"
	"hash/fnv"
	"slices"
	"strings"
	"unicode"

	"github.com/Nadim147c/go-mpris"
)

const (
	MaxIDLength = 150
	PrefixSize  = len(mpris.BaseInterface) + 1
)

func computeID(p *mpris.Player, m *Metadata) string {
	var buf bytes.Buffer

	playerName := p.GetName()
	buf.WriteString(stripPlayerName(playerName))
	buf.WriteByte('-')

	urlStr := removeUnwantedURLParameters(m.URL)
	hash := hashParts(playerName, m.RawArtist, m.RawTitle, urlStr)
	buf.Write(hash)

	buf.Grow(len(m.Title))

	lastMinus := true
	buf.WriteByte('-')
	for _, r := range m.Title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(r)
			lastMinus = false
		} else if !lastMinus {
			buf.WriteByte('-')
			lastMinus = true
		}
	}

	if lastMinus {
		buf.UnreadByte() //nolint
	}

	buf.Truncate(min(MaxIDLength, buf.Len()))
	return buf.String()
}

var encoder = base32.
	NewEncoding("abcdefghijklmnopqrstuvwxyz234567").
	WithPadding(base32.NoPadding)

// hashParts return fnv128a hashParts for given string.
func hashParts(v ...string) []byte {
	h := fnv.New128a()

	switch len(v) {
	case 0:
		panic("nothing to hash")
	case 1:
		h.Write([]byte(v[0]))
	default:
		lastIndex := len(v) - 1
		for i := range lastIndex {
			h.Write([]byte(v[i]))
			h.Write([]byte{':'})
		}
		h.Write([]byte(v[lastIndex]))
	}

	hash := h.Sum(nil)
	buf := make([]byte, encoder.EncodedLen(len(hash)))
	encoder.Encode(buf, hash)
	return buf
}

func isNonZero[T comparable](v T) bool {
	var zero T
	return zero != v
}

func hasAlbumAndArtists(p *mpris.Player) bool {
	album, err := p.GetAlbum()
	if err != nil || album == "" {
		return false
	}
	artists, err := p.GetArtist()
	if err != nil || len(artists) == 0 {
		return false
	}
	return slices.ContainsFunc(artists, isNonZero)
}

func removeUnwantedURLParameters(u *URL) string {
	host := u.Hostname()

	if strings.HasSuffix(host, "youtube.com") {
		query := u.Query()
		for k := range query {
			if k != "v" { // delete all key except v=<id>
				delete(query, k)
			}
		}
		u.RawQuery = query.Encode()
		return u.String()
	}

	if strings.HasSuffix(host, "spotify.com") {
		u.RawQuery = ""
		return u.String()
	}

	return u.String()
}
