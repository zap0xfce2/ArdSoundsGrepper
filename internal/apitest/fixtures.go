package apitest

import (
	"fmt"
	"strings"
)

const (
	MimiURN            = "urn:ard:show:daebe2a39366d28a"
	MimiNumericID      = "16240993"
	MimiPublishedCount = 247 // 250 Nodes − 2 unveröffentlichte − 1 Duplikat
	SmallURN           = "urn:ard:show:5a11"
	BrokenURN          = "urn:ard:show:b0b"
	fillerCount        = 241
)

type audioJSON struct {
	URL           string  `json:"url"`
	DownloadURL   *string `json:"downloadUrl"`
	AllowDownload bool    `json:"allowDownload"`
}

type nodeJSON struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	PublishDate string      `json:"publishDate"`
	Duration    int         `json:"duration"`
	IsPublished bool        `json:"isPublished"`
	Audios      []audioJSON `json:"audios"`
}

type programSetJSON struct {
	ID     string `json:"id"`
	CoreID string `json:"coreId"`
	Title  string `json:"title"`
	Items  struct {
		Nodes []nodeJSON `json:"nodes"`
	} `json:"items"`
}

type programSetResponse struct {
	Data struct {
		ProgramSet *programSetJSON `json:"programSet"`
	} `json:"data"`
}

type hitJSON struct {
	CoreID             string `json:"coreId"`
	Title              string `json:"title"`
	NumberOfElements   int    `json:"numberOfElements"`
	PublicationService struct {
		Title string `json:"title"`
	} `json:"publicationService"`
}

type searchResponse struct {
	Data struct {
		Search struct {
			ProgramSets struct {
				Nodes []hitJSON `json:"nodes"`
			} `json:"programSets"`
		} `json:"search"`
	} `json:"data"`
}

func (s *Server) buildProgramSet(id string, first, offset int) programSetResponse {
	var resp programSetResponse
	show := s.find(id)
	if show == nil {
		return resp
	}
	set := &programSetJSON{ID: show.NumericID, CoreID: show.URN, Title: show.Title}
	set.Items.Nodes = []nodeJSON{}
	for _, n := range page(show.Nodes, first, offset) {
		set.Items.Nodes = append(set.Items.Nodes, s.nodeJSON(n))
	}
	resp.Data.ProgramSet = set
	return resp
}

func page(nodes []Node, first, offset int) []Node {
	if offset >= len(nodes) {
		return nil
	}
	return nodes[offset:min(offset+first, len(nodes))]
}

// Wie die echte API: ein Audio mit Download-Freigabe, ein zweites ohne.
func (s *Server) nodeJSON(n Node) nodeJSON {
	path := cdnPrefix + n.ID + ".mp3"
	if n.Broken {
		path = htmlPath
	}
	url := s.URL + path
	primary := audioJSON{URL: url, AllowDownload: !n.Blocked}
	if !n.Blocked {
		primary.DownloadURL = &url
	}
	return nodeJSON{
		ID: n.ID, Title: n.Title, PublishDate: n.Date + "T00:01:00+02:00", Duration: durationSeconds,
		IsPublished: !n.Unpublished, Audios: []audioJSON{primary, {URL: url + ".alt"}},
	}
}

func (s *Server) buildSearch(term string) searchResponse {
	var resp searchResponse
	resp.Data.Search.ProgramSets.Nodes = []hitJSON{}
	for _, show := range s.shows {
		if !strings.Contains(strings.ToLower(show.Title), strings.ToLower(term)) {
			continue
		}
		hit := hitJSON{CoreID: show.URN, Title: show.Title, NumberOfElements: len(show.Nodes)}
		hit.PublicationService.Title = show.Sender
		resp.Data.Search.ProgramSets.Nodes = append(resp.Data.Search.ProgramSets.Nodes, hit)
	}
	return resp
}

// MimiShow bildet die echte Sendung samt aller Sonderfälle nach (250 Nodes, Offset-Paginierung nötig).
func MimiShow() Show {
	nodes := []Node{
		{ID: "11271873", Title: "Mikas Spielzeugkiste: Der neidische Bus", Date: "2026-10-02", Unpublished: true},
		{ID: "11043011", Title: "Im Kuscheltierkindergarten: Rückwärts gehen", Date: "2026-09-25"},
		{ID: "11043011", Title: "Im Kuscheltierkindergarten: Rückwärts gehen", Date: "2026-09-25"},
		{ID: "11043007", Title: "Mio/das Eichhörnchen: Der Pilz?", Date: "2026-09-18"},
		{ID: "16311645", Title: "Eichhörnchen Mio: Der Hügel auf der Wiese", Date: "2026-04-24"},
		{ID: "16251189", Title: "Eichhörnchen Mio: Der Hügel auf der Wiese", Date: "2026-04-24", Unpublished: true},
		{ID: "99999999", Title: "Gesperrte Folge", Date: "2026-04-20", Blocked: true},
		{ID: "88888888", Title: "Kaputte Folge", Date: "2026-04-19", Broken: true},
		{ID: "16251043", Title: "Mimi Sandmädchen - Trailer", Date: "2026-04-14"},
	}
	for i := range fillerCount {
		nodes = append(nodes, Node{
			ID: fmt.Sprintf("2%07d", i), Title: fmt.Sprintf("Füllfolge %03d", i),
			Date: fmt.Sprintf("2025-%02d-%02d", 1+i%12, 1+i%28),
		})
	}
	return Show{URN: MimiURN, NumericID: MimiNumericID, Title: "Mimi Sandmädchen", Sender: "rbb", Nodes: nodes}
}

// SmallShow: zwei ladbare Folgen, eine gesperrte, ein Trailer.
func SmallShow() Show {
	return Show{URN: SmallURN, NumericID: "5011", Title: "Kleine Sendung", Sender: "rbb", Nodes: []Node{
		{ID: "s3", Title: "Folge Drei", Date: "2026-09-25"},
		{ID: "s2", Title: "Folge Zwei", Date: "2026-09-18"},
		{ID: "s1", Title: "Folge Eins", Date: "2026-09-11", Blocked: true},
		{ID: "s0", Title: "Kleine Sendung - Trailer", Date: "2026-09-01"},
	}}
}

// BrokenShow: einzige Folge liefert HTML statt MP3.
func BrokenShow() Show {
	return Show{URN: BrokenURN, NumericID: "5012", Title: "Kaputte Sendung", Sender: "hr", Nodes: []Node{
		{ID: "b1", Title: "Kaputt", Date: "2026-09-25", Broken: true},
	}}
}
