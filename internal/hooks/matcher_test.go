package hooks

import "testing"

func TestMatchTool(t *testing.T) {
	cases := []struct {
		name    string
		matcher string
		tool    string
		want    bool
	}{
		{"empty matches all", "", "write", true},
		{"exact", "write", "write", true},
		{"miss", "write", "edit", false},
		{"star", "*", "anything", true},
		{"prefix", "task_*", "task_create", true},
		{"prefix miss", "task_*", "todo_write", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchTool(tc.matcher, tc.tool); got != tc.want {
				t.Errorf("matchTool(%q,%q)=%v want %v", tc.matcher, tc.tool, got, tc.want)
			}
		})
	}
}
