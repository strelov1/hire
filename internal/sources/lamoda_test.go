package sources

import (
	"context"
	"strings"
	"testing"
)

// lamodaListPage builds one offset list page: the items live under "data", paging meta
// under "meta".
func lamodaListPage(start, limit, total int, items ...string) string {
	return `{"data":[` + strings.Join(items, ",") +
		`],"meta":{"start":` + itoa(start) + `,"limit":` + itoa(limit) + `,"total":` + itoa(total) + `}}`
}

func lamodaListItem(id int, name, slug, location string) string {
	return `{"id":` + itoa(id) + `,"name":"` + name + `","slug":"` + slug +
		`","externalPublicationDate":"2026-06-11T16:04:39.000Z","location":{"name":"` + location + `"}}`
}

// lamodaDetail builds a detail response; the four body fields are assembled into the
// description.
func lamodaDetail(name, slug, introduction, duties, requirements, conditions string) string {
	return `{"data":{"attributes":{` +
		`"name":"` + name + `","slug":"` + slug + `",` +
		`"externalPublicationDate":"2026-06-11T16:04:39.000Z",` +
		`"introduction":"` + introduction + `","duties":"` + duties + `",` +
		`"requirements":"` + requirements + `","conditions":"` + conditions + `"}}}`
}

func TestLamodaProvider(t *testing.T) {
	if got := NewLamoda(nil).Provider(); got != "lamoda" {
		t.Errorf("Provider() = %q, want %q", got, "lamoda")
	}
}

func TestLamodaIsBoardless(t *testing.T) {
	if _, ok := NewLamoda(nil).(boardless); !ok {
		t.Error("lamoda should implement the boardless marker")
	}
}

func TestLamodaFetchPaginatesAndFetchesDetail(t *testing.T) {
	// total=3 with limit-based offset paging: start=0 (limit 2) then start=2.
	fake := (&routedHTTP{}).
		route("start%5D=0", lamodaListPage(0, 2, 3,
			lamodaListItem(2466, "Менеджер", "moskva/menedzher-2466", "Москва"),
			lamodaListItem(2467, "Аналитик", "spb/analitik-2467", "Удалённо"),
		)).
		route("start%5D=2", lamodaListPage(2, 2, 3,
			lamodaListItem(2468, "Инженер", "moskva/inzhener-2468", "Москва"),
		)).
		route("/vacancies/2466", lamodaDetail("Менеджер", "moskva/menedzher-2466",
			"<p>Intro.</p>", "<p>Duties.</p><script>alert(1)</script>", "<ul><li>Req</li></ul>", "<p>Cond.</p>")).
		route("/vacancies/2467", lamodaDetail("Аналитик", "spb/analitik-2467", "", "<p>Analyze.</p>", "", "")).
		route("/vacancies/2468", lamodaDetail("Инженер", "moskva/inzhener-2468", "", "<p>Engineer.</p>", "", ""))

	jobs, err := NewLamoda(fake).Fetch(context.Background(), CompanyEntry{Company: "Lamoda", Provider: "lamoda"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("len(jobs) = %d, want 3 across two pages", len(jobs))
	}

	byID := map[string]Job{}
	for _, j := range jobs {
		byID[j.ExternalID] = j
	}

	j, ok := byID["2466"]
	if !ok {
		t.Fatal("vacancy 2466 missing")
	}
	if j.Title != "Менеджер" {
		t.Errorf("Title = %q", j.Title)
	}
	if j.Company != "Lamoda" {
		t.Errorf("Company = %q, want Lamoda", j.Company)
	}
	if want := "https://job.lamoda.ru/vacancies/moskva/menedzher-2466"; j.URL != want {
		t.Errorf("URL = %q, want %q", j.URL, want)
	}
	if j.Location != "Москва" {
		t.Errorf("Location = %q, want list location.name", j.Location)
	}
	for _, want := range []string{"Intro.", "Duties.", "Req", "Cond."} {
		if !strings.Contains(j.Description, want) {
			t.Errorf("Description missing %q, got %q", want, j.Description)
		}
	}
	if strings.Contains(j.Description, "<script>") {
		t.Errorf("Description not sanitized: %q", j.Description)
	}
	if j.Remote {
		t.Error("2466 Remote = true, want false (Москва)")
	}
	if j.PostedAt == nil || j.PostedAt.Year() != 2026 {
		t.Errorf("PostedAt = %v, want parsed externalPublicationDate", j.PostedAt)
	}

	if !byID["2467"].Remote {
		t.Error("2467 Remote = false, want true (Удалённо location)")
	}
}

func TestLamodaDetailFallsBackToCommonWhenStructuredFieldsEmpty(t *testing.T) {
	// Retail vacancies leave the four structured fields empty and carry the whole body in
	// the "common" HTML attribute instead; the adapter falls back to it.
	detail := `{"data":{"attributes":{` +
		`"name":"Менеджер ПВЗ","slug":"lipetsk/menedzher-2414",` +
		`"externalPublicationDate":"2026-06-11T16:04:39.000Z",` +
		`"introduction":"","duties":"","requirements":"","conditions":"",` +
		`"common":"<p>Работа в пункте выдачи заказов.</p>"}}}`

	fake := (&routedHTTP{}).
		route("start%5D=0", lamodaListPage(0, 2, 1,
			lamodaListItem(2414, "Менеджер ПВЗ", "lipetsk/menedzher-2414", "Липецк"),
		)).
		route("/vacancies/2414", detail)

	jobs, err := NewLamoda(fake).Fetch(context.Background(), CompanyEntry{Company: "Lamoda", Provider: "lamoda"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	if !strings.Contains(jobs[0].Description, "Работа в пункте выдачи заказов.") {
		t.Errorf("Description = %q, want the common-field body", jobs[0].Description)
	}
}

func TestLamodaFetchSkipsFailedDetail(t *testing.T) {
	// 2467 has no detail route -> its detail fetch errors and the posting is skipped,
	// but 2466 still comes through.
	fake := (&routedHTTP{}).
		route("start%5D=0", lamodaListPage(0, 2, 2,
			lamodaListItem(2466, "Kept", "moskva/kept-2466", "Москва"),
			lamodaListItem(2467, "Broken", "moskva/broken-2467", "Москва"),
		)).
		route("/vacancies/2466", lamodaDetail("Kept", "moskva/kept-2466", "", "<p>ok</p>", "", ""))

	jobs, err := NewLamoda(fake).Fetch(context.Background(), CompanyEntry{Company: "Lamoda", Provider: "lamoda"})
	if err != nil {
		t.Fatalf("Fetch should not abort on one failed detail: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ExternalID != "2466" {
		t.Fatalf("want only 2466 to survive, got %d jobs", len(jobs))
	}
}

func TestLamodaEmptyListYieldsNoJobsNoError(t *testing.T) {
	fake := (&routedHTTP{}).route("start%5D=0", lamodaListPage(0, 2, 0))

	jobs, err := NewLamoda(fake).Fetch(context.Background(), CompanyEntry{Company: "Lamoda", Provider: "lamoda"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("got %d jobs, want 0", len(jobs))
	}
}
