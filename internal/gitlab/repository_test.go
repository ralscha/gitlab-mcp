package gitlab

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetFilePreservesBinaryContentThroughJSON(t *testing.T) {
	content := []byte{0, 0xff, 0xfe, 'a', '\n'}
	encoded := base64.StdEncoding.EncodeToString(content)
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(RepositoryFile{FileName: "binary.dat", Size: len(content), Encoding: "base64", Content: encoded})
	})
	file, err := client.GetFile(t.Context(), "1", "binary.dat", "main")
	if err != nil {
		t.Fatal(err)
	}
	if file.Encoding != "base64" {
		t.Fatalf("Encoding = %q, want base64", file.Encoding)
	}
	// Exercise the JSON boundary where invalid UTF-8 used to be replaced.
	payload, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip FileContent
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(roundTrip.Content)
	if err != nil || !bytes.Equal(decoded, content) {
		t.Fatalf("round-trip content = %v, err = %v", decoded, err)
	}
}

func TestCreateCommitPreservesEmptyAndOmittedContent(t *testing.T) {
	empty := ""
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Actions []map[string]json.RawMessage `json:"actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(body.Actions) != 5 {
			t.Errorf("actions = %#v", body.Actions)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for i, action := range body.Actions {
			content, present := action["content"]
			if i < 3 {
				if !present || string(content) != `""` {
					t.Errorf("action %d content = %s, present = %v", i, content, present)
				}
			} else if present {
				t.Errorf("action %d unexpectedly includes content %s", i, content)
			}
		}
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	})
	_, err := client.CreateCommit(t.Context(), "1", CreateCommitInput{
		Branch: "main", CommitMessage: "empty files",
		Actions: []CommitAction{
			{Action: "create", FilePath: "empty.txt", Content: &empty},
			{Action: "update", FilePath: "truncate.txt", Content: &empty},
			{Action: "move", FilePath: "empty-move.txt", PreviousPath: "source.txt", Content: &empty},
			{Action: "move", FilePath: "preserved-move.txt", PreviousPath: "other.txt"},
			{Action: "delete", FilePath: "deleted.txt"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateCommitValidatesActionRequirementsBeforeRequest(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("invalid action reached GitLab")
		w.WriteHeader(http.StatusBadRequest)
	})
	for _, test := range []struct {
		action CommitAction
		field  string
	}{
		{CommitAction{Action: "create", FilePath: "a"}, "content"},
		{CommitAction{Action: "update", FilePath: "a"}, "content"},
		{CommitAction{Action: "move", FilePath: "a"}, "previous_path"},
		{CommitAction{Action: "chmod", FilePath: "a"}, "execute_filemode"},
	} {
		t.Run(test.action.Action, func(t *testing.T) {
			_, err := client.CreateCommit(t.Context(), "1", CreateCommitInput{Actions: []CommitAction{test.action}})
			if err == nil || !strings.Contains(err.Error(), "requires "+test.field) {
				t.Fatalf("error = %v, want missing %s", err, test.field)
			}
		})
	}
}

func TestCreateCommitValidatesBase64WithoutSending(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("invalid content reached GitLab")
		w.WriteHeader(http.StatusBadRequest)
	})
	for _, test := range []struct {
		name    string
		content string
		large   bool
	}{
		{name: "invalid", content: "not base64!"},
		{name: "oversized", content: strings.Repeat("AAAA", MaxFileBytes/3+1), large: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := client.CreateCommit(t.Context(), "1", CreateCommitInput{Actions: []CommitAction{
				{Action: "create", FilePath: "a", Encoding: "base64", Content: &test.content},
			}})
			if test.large {
				if !errors.Is(err, ErrTooLarge) {
					t.Fatalf("error = %v, want ErrTooLarge", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "invalid base64") {
				t.Fatalf("error = %v, want invalid base64", err)
			}
		})
	}
}
