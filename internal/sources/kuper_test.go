package sources

import (
	"context"
	"strings"
	"testing"
)

// kuperPage builds one page of the result-block response: a pagination block carrying
// pages/totalCount and a vacancies block carrying the vacancy array.
func kuperPage(pages int, vacancies ...string) string {
	return `{"result":[` +
		`{"category":"pagination","data":{"totalCount":99,"pages":` + itoa(pages) + `}},` +
		`{"category":"vacancies","data":[` + strings.Join(vacancies, ",") + `]}` +
		`]}`
}

func kuperVacancy(id, title, city, friendlyURL string, idForURL int, description string, wf ...string) string {
	q := make([]string, len(wf))
	for i, w := range wf {
		q[i] = `"` + w + `"`
	}
	return `{"id":"` + id + `","title":"` + title + `","city":"` + city +
		`","friendlyUrl":"` + friendlyURL + `","idForUrl":` + itoa(idForURL) +
		`,"description":"` + description + `","wf":[` + strings.Join(q, ",") + `]}`
}

func TestKuperProvider(t *testing.T) {
	if got := NewKuper(nil).Provider(); got != "kuper" {
		t.Errorf("Provider() = %q, want %q", got, "kuper")
	}
}

func TestKuperIsBoardless(t *testing.T) {
	if _, ok := NewKuper(nil).(boardless); !ok {
		t.Error("kuper should implement the boardless marker")
	}
}

func TestKuperFetchPaginatesAndMaps(t *testing.T) {
	// pages=2 -> page 1 then page 2.
	fake := (&routedHTTP{}).
		route("page=1", kuperPage(2,
			kuperVacancy("uuid-1", "Курьер", "Москва", "Kurer-v-Kuper", 1272,
				"<p>Deliver things.</p><script>alert(1)</script>", "Удалённый"),
		)).
		route("page=2", kuperPage(2,
			kuperVacancy("uuid-2", "Оператор", "Орел", "Operator-v-Kuper", 1273,
				"<p>Answer calls.</p>", "Офисный"),
		))

	jobs, err := NewKuper(fake).Fetch(context.Background(), CompanyEntry{Company: "Kuper", Provider: "kuper"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2 across two pages", len(jobs))
	}

	byID := map[string]Job{}
	for _, j := range jobs {
		byID[j.ExternalID] = j
	}

	j, ok := byID["uuid-1"]
	if !ok {
		t.Fatal("vacancy uuid-1 missing")
	}
	if j.Title != "Курьер" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Company != "Kuper" {
		t.Errorf("Company = %q, want Kuper", j.Company)
	}
	if want := "https://kuper.ru/rabota/Kurer-v-Kuper-1272"; j.URL != want {
		t.Errorf("URL = %q, want %q", j.URL, want)
	}
	if j.Location != "Москва" {
		t.Errorf("Location = %q, want city", j.Location)
	}
	if !strings.Contains(j.Description, "Deliver things.") || strings.Contains(j.Description, "<script>") {
		t.Errorf("Description not sanitized/assembled: %q", j.Description)
	}
	if !j.Remote {
		t.Error("Remote = false, want true (Удалённый work format)")
	}
	if j.PostedAt != nil {
		t.Errorf("PostedAt = %v, want nil (no date in the source)", j.PostedAt)
	}

	if byID["uuid-2"].Remote {
		t.Error("uuid-2 Remote = true, want false (Офисный)")
	}
}

func TestKuperEmptyVacanciesYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("page=1", kuperPage(1))

	jobs, err := NewKuper(fake).Fetch(context.Background(), CompanyEntry{Company: "Kuper", Provider: "kuper"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}
