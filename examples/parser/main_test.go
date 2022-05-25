package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bolom009/web"
)

func Test_validate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		urls    []string
		wantErr bool
	}{
		{
			name:    "#1: Should return error on empty data",
			urls:    []string{},
			wantErr: true,
		},
		{
			name:    "#2: Should return error on second wrong url name",
			urls:    []string{"http://localhost:8080", "someurl"},
			wantErr: true,
		},
		{
			name:    "#3: Should return nil error",
			urls:    []string{"http://localhost:8080", "http://localhost:9090"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate(ctx, tt.urls); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getLenFromRequest(t *testing.T) {
	ctx := context.Background()
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Test ok"))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Test error"))
		}
	}))
	defer svr.Close()

	tests := []struct {
		name    string
		url     string
		want    int
		wantErr bool
	}{
		{
			name:    "#1: Should return length content = 4",
			url:     svr.URL + "/ok",
			want:    7,
			wantErr: false,
		},
		{
			name:    "#2: Should return error",
			url:     svr.URL + "/error",
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLenFromRequest(ctx, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLenFromRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getLenFromRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUrlsLen(t *testing.T) {
	ctx := context.Background()
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte("Test user path"))
		case "/group":
			_, _ = w.Write([]byte("Test group path"))
		}
	}))
	defer svr.Close()

	tests := []struct {
		name    string
		urls    []string
		want    string
		wantErr bool
	}{
		{
			name:    "#1: Should return correct values for urls",
			urls:    []string{svr.URL + "/user", svr.URL + "/group"},
			want:    "14\n15",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getUrlsLen(ctx, tt.urls)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUrlsLen() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if sortString(got) != sortString(tt.want) {
				t.Errorf("getUrlsLen() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parserUrlsHandler(t *testing.T) {
	webServer := web.NewHttpServer()
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Test user path"))
		case "/group":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Test group path"))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Test error"))
		}
	}))
	defer svr.Close()

	tests := []struct {
		name     string
		content  string
		code     int
		response string
		wantErr  bool
	}{
		{
			name:     "#1: Should return correct values for urls",
			content:  fmt.Sprintf(`{"urls": "%v\n%v"}`, svr.URL+"/user", svr.URL+"/group"),
			code:     http.StatusOK,
			response: "\"14\\n15\"",
			wantErr:  false,
		},
		{
			name:     "#2: Should return error on wrong bind type",
			content:  fmt.Sprintf(`1`),
			code:     http.StatusInternalServerError,
			response: "\"failed to bind body\"",
			wantErr:  true,
		},
		{
			name:     "#3: Should return error on empty urls value",
			content:  fmt.Sprintf(`{}`),
			code:     http.StatusBadRequest,
			response: "\"passed empty parameter urls\"",
			wantErr:  true,
		},
		{
			name:     "#3: Should return status ok but skip one broken url",
			content:  fmt.Sprintf(`{"urls": "%v\n%v"}`, svr.URL+"/user", svr.URL+"/error"),
			code:     http.StatusOK,
			response: "\"14\"",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.content))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			ctx := webServer.NewContext(req, rec)

			err := parserUrlsHandler(ctx)
			if err != nil {
				webServer.HTTPErrorHandler(err, ctx)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("parserUrlsHandler() error = %v, wantErr %v", err, tt.wantErr)
			}

			if rec.Code != tt.code {
				t.Errorf("parserUrlsHandler() got = %v, want %v", rec.Code, tt.code)
			}

			d, _ := ioutil.ReadAll(rec.Body)
			if sortString(string(d)) != sortString(tt.response) {
				t.Errorf("parserUrlsHandler() got = %v, want %v", string(d), tt.response)
			}
		})
	}
}

func sortString(w string) string {
	s := strings.Split(w, "")
	sort.Strings(s)
	return strings.Join(s, "")
}

func Test_delEmptyElems(t *testing.T) {
	tests := []struct {
		name string
		elms []string
		want []string
	}{
		{
			name: "#1: Should return corrected result",
			elms: []string{"1", "", "2", "", "3"},
			want: []string{"1", "2", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := delEmptyElems(tt.elms); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("delEmptyElems() = %v, want %v", got, tt.want)
			}
		})
	}
}
