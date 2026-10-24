package deck

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskReaderAllowsOnlyTaskMarkdownAndRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "tasks"), 0700)
	os.WriteFile(filepath.Join(root, "tasks", "safe.md"), []byte("# Safe task"), 0600)
	os.WriteFile(filepath.Join(root, "secret.md"), []byte("not readable"), 0600)
	a := testApp(&fakeHerdr{})
	a.root = root
	result, err := a.readExtension(httptest.NewRequest("GET", "/api/tasks/read?file=tasks/safe.md", nil))
	if err != nil || result.(map[string]string)["text"] != "# Safe task" {
		t.Fatalf("safe read: %v %v", result, err)
	}
	for _, file := range []string{"tasks/../secret.md", "secret.md", "tasks/safe.txt", `tasks/..\secret.md`} {
		_, err = a.readExtension(httptest.NewRequest("GET", "/api/tasks/read?file="+file, nil))
		if err == nil {
			t.Fatalf("accepted %s", file)
		}
	}
}
func TestDocsAllowlistDoesNotExposeOtherFiles(t *testing.T) {
	a := testApp(&fakeHerdr{})
	for _, name := range []string{"auth.json", "../agents", "CLAUDE", ""} {
		_, err := a.readExtension(httptest.NewRequest("GET", "/api/docs?name="+name, nil))
		if err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}
func TestLessonAppendIsNumberedAndKeepsExistingContent(t *testing.T) {
	a := testApp(&fakeHerdr{})
	a.root = t.TempDir()
	path := filepath.Join(a.root, "LESSONS.md")
	os.WriteFile(path, []byte("# Lessons\n\n4. Existing fact\n"), 0600)
	w := request(a, "POST", "/api/lessons/append", `{"text":"New fact\nwith detail"}`, "test-token", "")
	if w.Code != 200 {
		t.Fatalf("append failed: %s", w.Body.String())
	}
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "# Lessons\n\n4. Existing fact\n") || !strings.Contains(string(data), "5. New fact\n    with detail") {
		t.Fatalf("content changed: %s", data)
	}
}
