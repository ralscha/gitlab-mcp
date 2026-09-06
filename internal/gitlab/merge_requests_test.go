package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSetDraftTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		draft bool
		want  string
	}{
		{"mark draft", "Add feature", true, "Draft: Add feature"},
		{"avoid duplicate prefix", "draft: Add feature", true, "Draft: Add feature"},
		{"mark ready", "Draft: Add feature", false, "Add feature"},
		{"mark bracketed ready", "[Draft] Add feature", false, "Add feature"},
		{"mark legacy WIP ready", "WIP: Add feature", false, "Add feature"},
		{"preserve ready title", "Add feature", false, "Add feature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setDraftTitle(tt.title, tt.draft); got != tt.want {
				t.Errorf("setDraftTitle(%q, %v) = %q, want %q", tt.title, tt.draft, got, tt.want)
			}
		})
	}
}

func TestCreateMergeRequestMarksDraftThroughTitle(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["title"] != "Draft: Add feature" {
			t.Errorf("title = %#v", body["title"])
		}
		if _, present := body["draft"]; present {
			t.Errorf("unsupported draft field sent: %#v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"iid":2,"title":"Draft: Add feature","draft":true,"state":"opened"}`))
	})
	result, err := client.CreateMergeRequest(t.Context(), "1", CreateMergeRequestInput{
		SourceBranch: "feature", TargetBranch: "main", Title: "Add feature", Draft: true,
	})
	if err != nil || !result.Draft {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUpdateMergeRequestDraftStatusUsesCurrentTitle(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"iid":2,"title":"Add feature","state":"opened"}`))
		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["title"] != "Draft: Add feature" {
				t.Errorf("body = %#v", body)
			}
			if _, present := body["draft"]; present {
				t.Errorf("unsupported draft field sent: %#v", body)
			}
			_, _ = w.Write([]byte(`{"iid":2,"title":"Draft: Add feature","draft":true,"state":"opened"}`))
		default:
			t.Errorf("method = %s", r.Method)
		}
	})
	draft := true
	result, err := client.UpdateMergeRequest(t.Context(), "1", 2, UpdateMergeRequestInput{Draft: &draft})
	if err != nil || !result.Draft {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestListMergeRequestDiffs(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/api/v4/projects/group%2Fproject/merge_requests/4/diffs?page=2&per_page=100&unidiff=true" {
			t.Errorf("RequestURI = %q", r.RequestURI)
		}
		w.Header().Set("X-Next-Page", "3")
		_, _ = w.Write([]byte(`[{"old_path":"old.go","new_path":"new.go","diff":"@@ -1 +1 @@","renamed_file":true}]`))
	})
	diffs, page, err := client.ListMergeRequestDiffs(t.Context(), "group/project", 4, true, 2, 500)
	if err != nil || len(diffs) != 1 || !diffs[0].RenamedFile || page.PerPage != 100 || page.NextPage != 3 {
		t.Fatalf("diffs=%#v page=%#v err=%v", diffs, page, err)
	}
}

func TestAcceptMergeRequestUsesAutoMerge(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["auto_merge"] != true {
			t.Errorf("body = %#v", body)
		}
		if _, present := body["merge_when_pipeline_succeeds"]; present {
			t.Errorf("deprecated field sent: %#v", body)
		}
		_, _ = w.Write([]byte(`{"iid":2,"title":"Add feature","state":"opened"}`))
	})
	_, err := client.AcceptMergeRequest(t.Context(), "1", 2, AcceptMergeRequestInput{AutoMerge: true})
	if err != nil {
		t.Fatal(err)
	}
}
