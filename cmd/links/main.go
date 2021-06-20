package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"

	"github.com/anaskhan96/soup"
	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/shell"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver"
)

type orgLink struct {
	Link  string
	Title string
}

func (l orgLink) String() string {
	var result string
	if l.Link == "" {
		result = "<empty link>"
	} else {
		if l.Title != "" {
			result = fmt.Sprintf("* [[%s][%s]]", l.Link, l.Title)
		}
		result = fmt.Sprintf("* %s", l.Link)
	}
	return result
}

func acquireUrl() (*url.URL, error) {
	var uri *url.URL
	var err error
	urlCandidate, err := shell.ShellCmd("xsel -o -b", nil, nil, true, false)
	if err != nil {
		return nil, err
	}
	if len(*urlCandidate) > 0 {
		uri, err = url.ParseRequestURI(*urlCandidate)
		if err == nil {
			return uri, nil
		} else {
			ui.NotifyNormal("[scrape]", "Non-URL content in clipboard, trying active window name")
		}
	}

	windowName, err := xserver.GetCurrentWindowName(nil)
	if err != nil {
		return nil, err
	}
	uri, err = url.ParseRequestURI(*windowName)
	if err != nil {
		ui.NotifyCritical("[scrape]", "Non-URL content in active window name, giving up")
		return nil, impl.ErrInvalidUrl{*windowName}
	}

	return nil, impl.ErrInvalidUrl{}
}

func normalizeLink(link string, pageUrl *url.URL) (*url.URL, error) {
	if pageUrl == nil {
		return nil, fmt.Errorf("no parent page URL passed")
	}

	result := pageUrl
	uri, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "" && uri.Host == "" {
		result.Path = uri.Path
	} else {
		result = uri
	}

	return result, nil
}

func collectLinks(doc soup.Root, pageUrl *url.URL) (*string, []orgLink, error) {
	if pageUrl == nil {
		return nil, nil, fmt.Errorf("no parent page URL passed")
	}

	var title string
	titleNode := doc.Find("title")
	if titleNode.Error != nil {
		title = pageUrl.String()
	} else {
		title = titleNode.Text()
	}

	var links []orgLink
	linkNodes := doc.FindAll("a")
	for _, linkNode := range linkNodes {
		linkHref := linkNode.Attrs()["href"]
		linkUrl, err := normalizeLink(linkHref, pageUrl)
		if err != nil {
			continue
		}
		links = append(links, orgLink{Link: linkUrl.String(), Title: linkNode.Text()})
	}

	return &title, links, nil
}

func perform(ctx *cli.Context) error {
	pageUrl, err := acquireUrl()

	if err != nil {
		if e, ok := err.(impl.ErrInvalidUrl); ok {
			ui.NotifyCritical("[scrape]", fmt.Sprintf("Invalid URL: '%s'", e.Content))
			os.Exit(1)
		}
		return err
	}
	if pageUrl == nil {
		ui.NotifyCritical("[scrape]", "No page url provided")
		os.Exit(1)
	}
	ui.NotifyNormal("[scrape]", fmt.Sprintf("scraping from %s", pageUrl.String()))

	sessionName, err := ui.GetSelectionDmenu([]string{}, "save as", 1, ctx.String("selector-font"))
	pageSoup, err := soup.Get(pageUrl.String())
	if err != nil {
		return err
	}

	title, links, err := collectLinks(soup.HTMLParse(pageSoup), pageUrl)
	if err != nil {
		return err
	}

	orgContent := []string{
		fmt.Sprintf("#+TITLE: %s", title),
		fmt.Sprintf("#+PROPERTY: url %s", pageUrl.String()),
	}
	for _, link := range links {
		orgContent = append(orgContent, link.String())
	}

	file, err := os.OpenFile(fmt.Sprintf("%s/%s.org", ctx.String("export-path"), sessionName),
		os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, link := range orgContent {
		_, err = writer.WriteString(fmt.Sprintf("%s\n", link))
		if err != nil {
			return err
		}
	}
	ui.NotifyNormal("[scrape]", fmt.Sprintf("Scraped %d links", len(links)))

	return nil
}

func createCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "Links"
	app.Usage = "Collect links from page at provided URL"
	app.Description = "Links"
	app.Version = "0.0.1#master"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "export-path",
			Aliases:  []string{"p"},
			EnvVars:  []string{impl.EnvPrefix + "_DEFAULT_BROWSER_SESSIONS_STORE"},
			Usage:    "Path to export under",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "selector-font",
			Aliases:  []string{"f"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_FONT"},
			Usage:    "Font to use for selector application, e.g. dmenu, rofi, etc.",
			Required: false,
		},
	}
	app.Action = perform
	return app
}

func main() {
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
