// Package apitest stellt einen lokalen Mock der ARD-Audiothek-API samt CDN für Tests bereit.
package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	graphQLPath     = "/graphql"
	cdnPrefix       = "/cdn/"
	htmlPath        = "/cdn/html"
	durationSeconds = 383
)

// MP3Body ist ein minimaler Body, der den Magic-Byte-Check besteht.
func MP3Body() []byte {
	return append([]byte("ID3\x04\x00\x00\x00\x00\x00\x00"), bytes.Repeat([]byte{0xff, 0xfb}, 5000)...)
}

// Node ist ein Item einer Sendung, wie es die echte API liefert (inkl. Sonderfälle).
type Node struct {
	ID          string
	Title       string
	Date        string // YYYY-MM-DD
	Unpublished bool
	Blocked     bool // allowDownload:false
	Broken      bool // CDN liefert HTML statt MP3
}

type Show struct {
	URN       string
	NumericID string
	Title     string
	Sender    string
	Nodes     []Node
}

type Server struct {
	*httptest.Server
	shows []Show
}

func NewServer(t testing.TB, shows ...Show) *Server {
	t.Helper()
	s := &Server{shows: shows}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *Server) Endpoint() string { return s.URL + graphQLPath }

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == graphQLPath:
		s.handleGraphQL(w, r)
	case r.URL.Path == htmlPath:
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>nope</html>"))
	case strings.HasPrefix(r.URL.Path, cdnPrefix):
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(MP3Body())
	default:
		http.NotFound(w, r)
	}
}

type graphQLRequest struct {
	Query     string `json:"query"`
	Variables struct {
		ID     string `json:"id"`
		First  int    `json:"first"`
		Offset int    `json:"offset"`
		Query  string `json:"query"`
	} `json:"variables"`
}

func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.Contains(req.Query, "search(") {
		writeJSON(w, s.buildSearch(req.Variables.Query))
		return
	}
	writeJSON(w, s.buildProgramSet(req.Variables.ID, req.Variables.First, req.Variables.Offset))
}

func writeJSON[T any](w http.ResponseWriter, payload T) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) find(id string) *Show {
	for i := range s.shows {
		if s.shows[i].URN == id || s.shows[i].NumericID == id {
			return &s.shows[i]
		}
	}
	return nil
}
