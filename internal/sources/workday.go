package sources

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// workdayPageLimit is Workday's max listing page size; workdayDetailWorkers caps how
// many per-posting detail requests a single board issues concurrently.
const (
	workdayPageLimit     = 20
	workdayDetailWorkers = 8
)

// workday adapts Workday's public "CXS" careers API. The board id is the public board
// host and site path, e.g. "ringcentral.wd1.myworkdayjobs.com/RingCentral_Careers";
// the API tenant is the host's first label (here "ringcentral"). The listing endpoint
// is POST-only and carries no description, so it pages the postings and fetches each
// posting's detail (bounded-concurrency) to assemble the description.
type workday struct {
	http HTTPClient
}

// NewWorkday builds the Workday adapter over the given HTTP client.
func NewWorkday(c HTTPClient) Source { return workday{http: c} }

func (workday) Provider() string { return "workday" }

// workdayBoard is a configured board parsed into the parts the CXS endpoints need.
type workdayBoard struct {
	host, tenant, site string
}

// parseWorkdayBoard splits "host/site" (e.g. "acme.wd1.myworkdayjobs.com/Careers") into
// the host, the tenant (the host's first label), and the site.
func parseWorkdayBoard(board string) (workdayBoard, error) {
	host, site, ok := strings.Cut(board, "/")
	if !ok || host == "" || site == "" {
		return workdayBoard{}, fmt.Errorf("workday: board %q must be \"host/site\"", board)
	}
	tenant, _, ok := strings.Cut(host, ".")
	if !ok || tenant == "" {
		return workdayBoard{}, fmt.Errorf("workday: board host %q has no tenant label", host)
	}
	return workdayBoard{host: host, tenant: tenant, site: site}, nil
}

// workdayPosting is one item from the jobs listing (no description here).
type workdayPosting struct {
	Title         string `json:"title"`
	ExternalPath  string `json:"externalPath"`
	LocationsText string `json:"locationsText"`
}

func (s workday) Fetch(ctx context.Context, e CompanyEntry) ([]Job, error) {
	b, err := parseWorkdayBoard(e.Board)
	if err != nil {
		return nil, err
	}

	postings, err := s.listPostings(ctx, b)
	if err != nil {
		return nil, err
	}

	// Each posting's description comes from its own detail request, fanned out under a
	// bounded worker pool.
	return fetchDetails(postings, workdayDetailWorkers, func(p workdayPosting) (Job, bool) {
		return s.detail(ctx, e, b, p)
	}), nil
}

// listPostings pages through the board's postings via the POST-only jobs endpoint,
// stopping when a page is empty or all postings reported by total have been collected.
func (s workday) listPostings(ctx context.Context, b workdayBoard) ([]workdayPosting, error) {
	url := fmt.Sprintf("https://%s/wday/cxs/%s/%s/jobs", b.host, b.tenant, b.site)
	var postings []workdayPosting
	for offset := 0; ; {
		reqBody := map[string]any{
			"appliedFacets": map[string]any{},
			"limit":         workdayPageLimit,
			"offset":        offset,
			"searchText":    "",
		}
		var page struct {
			Total       int              `json:"total"`
			JobPostings []workdayPosting `json:"jobPostings"`
		}
		if err := s.http.PostJSON(ctx, url, reqBody, &page); err != nil {
			return nil, fmt.Errorf("workday: list board %s: %w", b.site, err)
		}
		if len(page.JobPostings) == 0 {
			break
		}
		postings = append(postings, page.JobPostings...)
		offset += len(page.JobPostings)
		if offset >= page.Total {
			break
		}
	}
	return postings, nil
}

// detail fetches one posting's detail and maps it to a Job, returning ok=false when the
// detail request fails so the caller can skip just that posting.
func (s workday) detail(ctx context.Context, e CompanyEntry, b workdayBoard, p workdayPosting) (Job, bool) {
	url := fmt.Sprintf("https://%s/wday/cxs/%s/%s%s", b.host, b.tenant, b.site, p.ExternalPath)

	var d struct {
		JobPostingInfo struct {
			Title          string `json:"title"`
			JobDescription string `json:"jobDescription"`
			Location       string `json:"location"`
			StartDate      string `json:"startDate"`
			ExternalURL    string `json:"externalUrl"`
			RemoteType     string `json:"remoteType"`
		} `json:"jobPostingInfo"`
	}
	if err := s.http.GetJSON(ctx, url, &d); err != nil {
		return Job{}, false
	}
	info := d.JobPostingInfo

	title := strings.TrimSpace(info.Title)
	if title == "" {
		title = strings.TrimSpace(p.Title)
	}
	location := info.Location
	if location == "" {
		location = p.LocationsText
	}
	jobURL := info.ExternalURL
	if jobURL == "" {
		jobURL = fmt.Sprintf("https://%s/%s%s", b.host, b.site, p.ExternalPath)
	}
	remote := isRemote(location) || strings.Contains(strings.ToLower(info.RemoteType), "remote")

	return Job{
		ExternalID:  p.ExternalPath,
		URL:         jobURL,
		Title:       title,
		Company:     e.Company,
		Location:    location,
		Description: sanitizeHTML(info.JobDescription),
		Remote:      remote,
		PostedAt:    parseWorkdayDate(info.StartDate),
	}, true
}

// parseWorkdayDate reads Workday's startDate, which may be a full RFC3339 timestamp or
// a date-only value, returning nil for anything unparseable (posted_at is nullable).
func parseWorkdayDate(s string) *time.Time {
	if t := parseRFC3339(s); t != nil {
		return t
	}
	return parseDate(s)
}
