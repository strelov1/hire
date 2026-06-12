package sources

import (
	"context"
	"fmt"
)

// workableBaseURL is the Workable public job-board widget API root.
const workableBaseURL = "https://apply.workable.com/api/v1/widget/accounts"

// workable adapts the Workable public widget API. With details=true the list endpoint
// carries an inline HTML description, so no per-posting detail request is needed.
type workable struct {
	http HTTPClient
}

// NewWorkable builds the Workable adapter over the given HTTP client.
func NewWorkable(c HTTPClient) Source { return workable{http: c} }

func (workable) Provider() string { return "workable" }

func (w workable) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	url := fmt.Sprintf("%s/%s?details=true", workableBaseURL, e.Board)

	var resp struct {
		Jobs []struct {
			Title         string `json:"title"`
			Shortcode     string `json:"shortcode"`
			URL           string `json:"url"`
			Description   string `json:"description"`
			PublishedOn   string `json:"published_on"`
			City          string `json:"city"`
			State         string `json:"state"`
			Country       string `json:"country"`
			Telecommuting bool   `json:"telecommuting"`
		} `json:"jobs"`
	}
	if err := w.http.GetJSON(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("workable: fetch board %s: %w", e.Board, err)
	}

	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{
			ExternalID:  j.Shortcode,
			URL:         j.URL,
			Title:       j.Title,
			Company:     e.Company,
			Location:    joinNonEmpty(j.City, j.State, j.Country),
			Description: sanitizeHTML(j.Description),
			Remote:      j.Telecommuting,
			PostedAt:    parseDate(j.PublishedOn),
		})
	}
	return jobs, nil
}
