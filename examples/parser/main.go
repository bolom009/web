package main

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bolom009/web/middleware"

	"github.com/bolom009/web"
)

type urlsRequest struct {
	Urls string `json:"urls"`
}

func parserUrlsHandler(c web.Context) error {
	var req urlsRequest
	if err := c.BindBody(&req); err != nil {
		return web.NewHTTPErrorWithInternal(err, http.StatusInternalServerError, "failed to bind body")
	}

	ctx := c.Request().Context()
	urls := delEmptyElems(strings.Split(strings.ReplaceAll(req.Urls, "\r\n", "\n"), "\n"))

	if err := validate(ctx, urls); err != nil {
		return err
	}

	resContentLens, err := getUrlsLen(ctx, urls)
	if err != nil {
		log.Println("FailedToGetUrlsLen", err)
		return web.NewHTTPErrorWithInternal(err, http.StatusInternalServerError, "failed to get urls content lens")
	}

	return c.JSON(http.StatusOK, resContentLens)
}

func validate(_ context.Context, urls []string) error {
	if len(urls) == 0 {
		return web.NewHTTPError(http.StatusBadRequest, "passed empty parameter urls")
	}

	for _, u := range urls {
		if _, err := url.ParseRequestURI(u); err != nil {
			return web.NewHTTPError(http.StatusBadRequest, "passed wrong value for url: "+u)
		}
	}

	return nil
}

func getUrlsLen(ctx context.Context, urls []string) (string, error) {
	var (
		resContentLens = make([]string, 0, len(urls))
		wg             = new(sync.WaitGroup)
	)

	// Processing urls with skip the error (only log), no rules for prevent processing if got error
	for _, u := range urls {
		wg.Add(1)

		copyUrl := u
		go func() {
			defer wg.Done()

			respLen, err := getLenFromRequest(ctx, copyUrl)
			if err != nil {
				log.Println("FailedToGetLenFromRequest", err)
				return
			}

			resContentLens = append(resContentLens, strconv.Itoa(respLen))
		}()
	}

	wg.Wait()

	return strings.Join(resContentLens, "\n"), nil
}

func getLenFromRequest(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	b, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, errors.New(string(b))
	}

	return len(string(b)), nil
}

func delEmptyElems(elms []string) []string {
	var r []string
	for _, str := range elms {
		if str != "" {
			r = append(r, str)
		}
	}

	return r
}

func main() {
	r := web.NewHttpServer()
	r.Use(middleware.RateLimiter(100, time.Minute)) // rate - 100/min
	r.Add(http.MethodPost, "/", parserUrlsHandler)
	if err := r.Start(":8080"); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
