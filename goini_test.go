package goini

import (
	"testing"
)

var config = Load("app.ini")

func TestGoini_Get(t *testing.T) {
	testCases := []struct {
		Key    string
		Expect string
	}{
		{
			Key:    "env",
			Expect: "test",
		},
		{
			Key:    "host",
			Expect: "127.0.0.1",
		},
		{
			Key:    "port",
			Expect: "8080",
		},
	}

	for _, v := range testCases {
		ret := config.Get(v.Key)
		if ret != v.Expect {
			t.Errorf("Goini: Not as expected ret=%v, expect=%v", ret, v.Expect)
		}
	}
}

func TestGoini_GetBySection(t *testing.T) {
	testCases := []struct {
		Key     string
		Section string
		Expect  string
	}{
		{
			Key:     "driver",
			Section: "db",
			Expect:  "mysql",
		},
		{
			Key:     "host",
			Section: "db",
			Expect:  "127.0.0.1",
		},
		{
			Key:     "port",
			Section: "db",
			Expect:  "3306",
		},
	}

	for _, v := range testCases {
		ret := config.Get(v.Key, v.Section)
		if ret != v.Expect {
			t.Errorf("Goini: Not as expected ret=%v, expect=%v", ret, v.Expect)
		}
	}
}
