package api

import "context"

const searchLimit = 20

const searchQuery = `query ($query: String, $limit: Int) {
  search(query: $query, type: ProgramSets, limit: $limit) {
    programSets { nodes { coreId title numberOfElements publicationService { title } } }
  }
}`

type ShowHit struct {
	URN          string
	Title        string
	Sender       string
	EpisodeCount int
}

type searchVars struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type searchData struct {
	Search struct {
		ProgramSets struct {
			Nodes []hitNode `json:"nodes"`
		} `json:"programSets"`
	} `json:"search"`
}

type hitNode struct {
	CoreID             string `json:"coreId"`
	Title              string `json:"title"`
	NumberOfElements   int    `json:"numberOfElements"`
	PublicationService *struct {
		Title string `json:"title"`
	} `json:"publicationService"`
}

// SearchShows sucht Sendungen; die API sucht unscharf, daher wählt immer der Nutzer.
func (c *Client) SearchShows(ctx context.Context, term string) ([]ShowHit, error) {
	var data searchData
	if err := post(ctx, c, searchQuery, searchVars{Query: term, Limit: searchLimit}, &data); err != nil {
		return nil, err
	}
	nodes := data.Search.ProgramSets.Nodes
	hits := make([]ShowHit, 0, len(nodes))
	for _, n := range nodes {
		hit := ShowHit{URN: n.CoreID, Title: n.Title, EpisodeCount: n.NumberOfElements}
		if n.PublicationService != nil {
			hit.Sender = n.PublicationService.Title
		}
		hits = append(hits, hit)
	}
	return hits, nil
}
