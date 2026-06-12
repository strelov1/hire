package sources

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// smartRecruitersBaseURL is the SmartRecruiters public postings API root.
const smartRecruitersBaseURL = "https://api.smartrecruiters.com/v1/companies"

// smartRecruitersPageLimit is the postings page size; smartRecruitersDetailWorkers caps
// how many per-posting detail requests a single board issues concurrently.
const (
	smartRecruitersPageLimit     = 100
	smartRecruitersDetailWorkers = 8
)

// smartRecruiters adapts the SmartRecruiters public API. Unlike the other adapters its
// list endpoint carries no description, so it paginates the postings and fetches each
// posting's detail (bounded-concurrency) to assemble the description.
type smartRecruiters struct {
	http HTTPClient
}

// NewSmartRecruiters builds the SmartRecruiters adapter over the given HTTP client.
func NewSmartRecruiters(c HTTPClient) Source { return smartRecruiters{http: c} }

func (smartRecruiters) Provider() string { return "smartrecruiters" }

// srSection is one HTML section of a posting's job ad (the description lives here).
type srSection struct {
	Text string `json:"text"`
}

// smartRecruitersPosting is one item from the postings list (no description here).
type smartRecruitersPosting struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ReleasedDate string `json:"releasedDate"`
	Location     struct {
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
		Remote  bool   `json:"remote"`
	} `json:"location"`
}

func (s smartRecruiters) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	postings, err := s.listPostings(ctx, e.Board)
	if err != nil {
		return nil, err
	}

	// Each posting's description comes from its own detail request. Fan out with a
	// bounded worker pool, writing each result into its own slot; a failed detail
	// leaves a nil slot that is dropped below, so one bad posting never aborts the board.
	jobs := make([]*Job, len(postings))
	sem := make(chan struct{}, smartRecruitersDetailWorkers)
	var wg sync.WaitGroup
	for i, p := range postings {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, p smartRecruitersPosting) {
			defer wg.Done()
			defer func() { <-sem }()
			if j, ok := s.detail(ctx, e, p); ok {
				jobs[i] = &j
			}
		}(i, p)
	}
	wg.Wait()

	out := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		if j != nil { // nil = detail fetch failed, skipped
			out = append(out, *j)
		}
	}
	return out, nil
}

// listPostings pages through the board's postings, stopping when a page is empty or all
// postings reported by totalFound have been collected.
func (s smartRecruiters) listPostings(ctx context.Context, board string) ([]smartRecruitersPosting, error) {
	var postings []smartRecruitersPosting
	for offset := 0; ; {
		url := fmt.Sprintf("%s/%s/postings?limit=%d&offset=%d", smartRecruitersBaseURL, board, smartRecruitersPageLimit, offset)
		var page struct {
			TotalFound int                      `json:"totalFound"`
			Content    []smartRecruitersPosting `json:"content"`
		}
		if err := s.http.GetJSON(ctx, url, &page); err != nil {
			return nil, fmt.Errorf("smartrecruiters: list board %s: %w", board, err)
		}
		if len(page.Content) == 0 {
			break
		}
		postings = append(postings, page.Content...)
		offset += len(page.Content)
		if offset >= page.TotalFound {
			break
		}
	}
	return postings, nil
}

// detail fetches one posting's detail and maps it to a Job, returning ok=false when the
// detail request fails so the caller can skip just that posting.
func (s smartRecruiters) detail(ctx context.Context, e CompanyEntry, p smartRecruitersPosting) (Job, bool) {
	url := fmt.Sprintf("%s/%s/postings/%s", smartRecruitersBaseURL, e.Board, p.ID)

	var d struct {
		PostingURL string `json:"postingUrl"`
		JobAd      struct {
			Sections struct {
				JobDescription        srSection `json:"jobDescription"`
				Qualifications        srSection `json:"qualifications"`
				AdditionalInformation srSection `json:"additionalInformation"`
			} `json:"sections"`
		} `json:"jobAd"`
	}
	if err := s.http.GetJSON(ctx, url, &d); err != nil {
		return Job{}, false
	}

	// companyDescription is intentionally excluded — it is boilerplate, not the role.
	sec := d.JobAd.Sections
	body := sec.JobDescription.Text + sec.Qualifications.Text + sec.AdditionalInformation.Text

	return Job{
		ExternalID:  p.ID,
		URL:         d.PostingURL,
		Title:       strings.TrimSpace(p.Name),
		Company:     e.Company,
		Location:    joinNonEmpty(p.Location.City, p.Location.Region, p.Location.Country),
		Description: sanitizeHTML(body),
		Remote:      p.Location.Remote,
		PostedAt:    parseRFC3339(p.ReleasedDate),
	}, true
}
