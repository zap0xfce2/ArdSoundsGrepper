package api

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	pageSize    = 100
	invalidDate = "0000-00-00"
)

// Das Datum landet im Dateinamen; ungeprüft wäre z. B. "../" aus der API ein Pfad-Traversal.
var datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// Cursor-Paginierung (after:) ist serverseitig kaputt, daher first + offset.
const episodesQuery = `query ($id: ID!, $first: Int, $offset: Int) {
  programSet(id: $id) {
    id coreId title numberOfElements
    items(first: $first, offset: $offset) {
      nodes {
        id title publishDate duration isPublished
        audios { url downloadUrl allowDownload mimeType }
      }
    }
  }
}`

type Episode struct {
	ID              string
	Title           string
	PublishDate     string
	DurationSeconds int
	DownloadURL     string // leer, wenn kein Download erlaubt ist
}

func (e Episode) Date() string {
	if date := datePrefix.FindString(e.PublishDate); date != "" {
		return date
	}
	return invalidDate
}

func (e Episode) IsTrailer() bool {
	return strings.Contains(strings.ToLower(e.Title), "trailer")
}

type Show struct {
	ID       string
	URN      string
	Title    string
	Episodes []Episode // veröffentlicht, ohne Duplikate, neueste zuerst
}

type episodeVars struct {
	ID     string `json:"id"`
	First  int    `json:"first"`
	Offset int    `json:"offset"`
}

type programSetData struct {
	ProgramSet *programSetNode `json:"programSet"`
}

type programSetNode struct {
	ID     string `json:"id"`
	CoreID string `json:"coreId"`
	Title  string `json:"title"`
	Items  struct {
		Nodes []itemNode `json:"nodes"`
	} `json:"items"`
}

type itemNode struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	PublishDate string      `json:"publishDate"`
	Duration    int         `json:"duration"`
	IsPublished bool        `json:"isPublished"`
	Audios      []audioNode `json:"audios"`
}

type audioNode struct {
	DownloadURL   string `json:"downloadUrl"`
	AllowDownload bool   `json:"allowDownload"`
}

// FetchShow lädt eine Sendung per URN oder numerischer ID mit allen veröffentlichten Folgen.
func (c *Client) FetchShow(ctx context.Context, id string) (Show, error) {
	var show Show
	var nodes []itemNode
	for offset := 0; ; offset += pageSize {
		var data programSetData
		if err := post(ctx, c, episodesQuery, episodeVars{ID: id, First: pageSize, Offset: offset}, &data); err != nil {
			return Show{}, err
		}
		if data.ProgramSet == nil {
			return Show{}, fmt.Errorf("%w: %s", ErrShowNotFound, id)
		}
		show.ID, show.URN, show.Title = data.ProgramSet.ID, data.ProgramSet.CoreID, data.ProgramSet.Title
		page := data.ProgramSet.Items.Nodes
		nodes = append(nodes, page...)
		if len(page) < pageSize {
			break
		}
	}
	show.Episodes = publishedEpisodes(nodes)
	return show, nil
}

// Die API liefert auch künftige und zurückgezogene Items (isPublished:false) – die fliegen raus.
func publishedEpisodes(nodes []itemNode) []Episode {
	seen := make(map[string]bool, len(nodes))
	episodes := make([]Episode, 0, len(nodes))
	for _, n := range nodes {
		if !n.IsPublished || seen[n.ID] {
			continue
		}
		seen[n.ID] = true
		episodes = append(episodes, Episode{
			ID: n.ID, Title: n.Title, PublishDate: n.PublishDate,
			DurationSeconds: n.Duration, DownloadURL: downloadURL(n.Audios),
		})
	}
	slices.SortStableFunc(episodes, func(a, b Episode) int { return strings.Compare(b.PublishDate, a.PublishDate) })
	return episodes
}

func downloadURL(audios []audioNode) string {
	for _, audio := range audios {
		if audio.AllowDownload && audio.DownloadURL != "" {
			return audio.DownloadURL
		}
	}
	return ""
}
