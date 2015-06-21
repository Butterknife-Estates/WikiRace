package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/satori/go.uuid"
)

const (
	randomPage       = "Special:Random"
	queryUrlTemplate = "https://%s.wikipedia.org/wiki/%s"
	lang             = "en"
)

var sessions map[string]*Session

func main() {
	sessions = map[string]*Session{}
	port := "8080"

	e := echo.New()

	// Middleware
	e.Use(mw.Logger())
	//e.Use(mw.Recover())
	e.Get("/rand", rand)
	e.Get("/start", begin)
	// Routes
	e.Get("/wiki/:sessionId/:page", hello)

	fmt.Println(fmt.Sprintf("Server started on port %s", port))

	e.Run(fmt.Sprintf(":%s", port))
}

func rand(c *echo.Context) error {
	title, err := getRandomPage()
	if err != nil {
		return err
	}
	return c.String(http.StatusOK, title)
}

func begin(c *echo.Context) error {
	begin, err := getRandomPage()
	end, err := getRandomPage()

	if err != nil {
		return err
	}
	session := &Session{Id: uuid.NewV4().String(), Begin: begin, End: end}

	sessions[session.Id] = session

	html, err := getPage(session, randomPage)

	if err != nil {
		return err
	}
	return c.HTML(http.StatusOK, html)
}

func hello(c *echo.Context) error {
	id, page := c.P(0), c.P(1)
	if session, exists := sessions[id]; exists {
		if session.End == page {
			return c.HTML(http.StatusOK, "YOU WIN!!!")
		}
		html, err := getPage(session, page)
		if err != nil {
			return err
		}
		return c.HTML(http.StatusOK, html)
	}
	return errors.New("session not found")
}

func getPage(session *Session, page string) (string, error) {
	doc, err := goquery.NewDocument(fmt.Sprintf(queryUrlTemplate, lang, url.QueryEscape(page)))
	if err != nil {
		return "", err
	}

	heading := doc.Find("#firstHeading")
	heading.PrependHtml(fmt.Sprintf("<h1>You are trying to find %s</h1>", session.End))
	content := doc.Find("#mw-content-text")

	content.PrependSelection(heading)

	content.Find(".thumb").Remove()
	content.Find("img").Remove()
	content.Find(".mw-editsection").Remove()
	content.Find("a[href^='http']").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithHtml(fmt.Sprintf("<b>%s</b>", s.Text()))
	})
	content.Find("a").Each(func(i int, s *goquery.Selection) {
		if v, exists := s.Attr("href"); exists {
			result := strings.Replace(v, "/wiki/", fmt.Sprintf("/wiki/%s/", session.Id), -1)
			s.SetAttr("href", result)
		}
	})
	return content.Html()
}

func getRandomPage() (string, error) {

	// Set up the HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf(queryUrlTemplate, lang, randomPage), nil)
	if err != nil {
		return "", err
	}

	transport := http.Transport{}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return "", errors.New("failed")
	}
	if parts := strings.Split(resp.Header.Get("Location"), "/"); len(parts) > 0 {
		return parts[len(parts)-1], nil
	}
	return "", errors.New("failed")
}

type Session struct {
	Id    string
	Begin string
	End   string
}
