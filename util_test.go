package goplugify

import (
	"encoding/json"
	"reflect"
	"testing"
)

type inner struct {
	Value string
}

type testStruct struct {
	Exported   string
	unexported int
	Inner      inner
}

func TestConvertArgument(t *testing.T) {
	type testCase struct {
		name         string
		input        any
		expectedType reflect.Type
		wantValue    any
		wantErr      bool
	}

	tests := []testCase{
		// --- basic type ---
		{"int to int", 123, reflect.TypeOf(int(0)), 123, false},
		{"float64 to int", 3.0, reflect.TypeOf(int(0)), 3, false},
		{"int to float64", 5, reflect.TypeOf(float64(0)), 5.0, false},
		{"string to string", "hello", reflect.TypeOf(""), "hello", false},
		{"bool to bool", true, reflect.TypeOf(true), true, false},

		// --- struct <-> struct ---
		{"struct to struct (same type)", testStruct{"a", 10, inner{"x"}}, reflect.TypeOf(testStruct{}), testStruct{"a", 10, inner{"x"}}, false},

		// --- map -> struct ---
		{"map to struct", map[string]any{"Exported": "hi", "unexported": 42, "Inner": map[string]any{"Value": "v"}}, reflect.TypeOf(testStruct{}), testStruct{"hi", 42, inner{"v"}}, false},

		// --- slice ---
		{"[]map to []struct", []map[string]any{{"Exported": "m1", "unexported": 1}, {"Exported": "m2", "unexported": 2}}, reflect.TypeOf([]testStruct{}),
			[]testStruct{{"m1", 1, inner{}}, {"m2", 2, inner{}}}, false},

		// --- pointer ---
		{"*struct to struct", &testStruct{"ptr", 88, inner{"v"}}, reflect.TypeOf(testStruct{}), testStruct{"ptr", 88, inner{"v"}}, false},
		{"struct to *struct", testStruct{"ptr2", 99, inner{"v2"}}, reflect.TypeOf(&testStruct{}), &testStruct{"ptr2", 99, inner{"v2"}}, false},

		// --- invalid ---
		{"struct to int (invalid)", testStruct{}, reflect.TypeOf(int(0)), nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertArgument(tt.input, tt.expectedType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotInterface := got.Interface()

			gotJSON, _ := json.Marshal(gotInterface)
			wantJSON, _ := json.Marshal(tt.wantValue)

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("\nname=%s\nexpected=%s\n     got=%s", tt.name, string(wantJSON), string(gotJSON))
			}
		})
	}
}

func TestConvertTo(t *testing.T) {
	type Book struct {
		Title  string
		Author string
		Pages  int
	}

	type BookDst struct {
		Title  string
		Author string
		Pages  int
	}

	booksSrc := []Book{
		{"The Go Programming Language", "Alan A. A. Donovan", 400},
		{"Introducing Go", "Caleb Doxsey", 124},
	}

	var booksDst []BookDst

	err := ConvertTo(booksSrc, &booksDst)
	if err != nil {
		t.Fatalf("ConvertTo failed: %v", err)
	}

	if len(booksDst) != len(booksSrc) {
		t.Fatalf("expected %d books, got %d", len(booksSrc), len(booksDst))
	}

	for i, book := range booksDst {
		if book.Title != booksSrc[i].Title || book.Author != booksSrc[i].Author || book.Pages != booksSrc[i].Pages {
			t.Errorf("book %d mismatch: expected %+v, got %+v", i, booksSrc[i], book)
		}
	}
}