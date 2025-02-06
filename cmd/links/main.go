package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"

	"github.com/anaskhan96/soup"
	"github.com/urfave/cli/v2"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/ui"
	"github.com/wiedzmin/toolbox/impl/xserver"
	"github.com/wiedzmin/toolbox/impl/xserver/xkb"
	"go.uber.org/zap"
)

var logger *zap.Logger

type orgLink struct {
	Link  string
	Title string
}

func (l orgLink) String() string {
	var result string
	if l.Link == "" {
		result = "<empty link>"
	} else {
		result = fmt.Sprintf("* %s", l.Link)
		if l.Title != "" {
			result = fmt.Sprintf("* [[%s][%s]]", l.Link, l.Title)
		}
	}
	return result
}

func acquireUrl() (*url.URL, error) {
	var uri *url.URL
	var err error
	l := logger.Sugar()
	urlCandidate, err := xserver.ReadClipboard(false)
	l.Debugw("[acquireUrl]", "urlCandidate", urlCandidate, "err", err)
	if err != nil {
		return nil, err
	}
	if len(*urlCandidate) > 0 {
		uri, err = url.ParseRequestURI(*urlCandidate)
		l.Debugw("[acquireUrl]", "uri (from XA_CLIPBOARD)", uri, "err", err)
		if err == nil {
			return uri, nil
		} else {
			ui.NotifyNormal("[scrape]", "Non-URL content in clipboard, trying active window name")
		}
	}

	x, err := xserver.NewX()
	if err != nil {
		return nil, err
	}
	windowTraits, err := x.GetWindowTraits(nil)
	l.Debugw("[acquireUrl]", "windowTraits", windowTraits, "err", err)
	if err != nil {
		return nil, err
	}
	uri, err = url.ParseRequestURI(windowTraits.Title)
	l.Debugw("[acquireUrl]", "uri (from window name)", uri, "err", err)
	if err != nil {
		ui.NotifyCritical("[scrape]", "Non-URL content in active window name, giving up")
		return nil, impl.ErrInvalidUrl{Content: windowTraits.Title}
	}

	return nil, impl.ErrInvalidUrl{}
}

func normalizeLink(link string, pageUrl *url.URL) (*url.URL, error) {
	l := logger.Sugar()
	if pageUrl == nil {
		return nil, fmt.Errorf("no parent page URL passed")
	}

	result := pageUrl
	uri, err := url.Parse(link)
	l.Debugw("[normalizeLink]", "parsed uri", uri, "err", err)
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
	l := logger.Sugar()
	if pageUrl == nil {
		return nil, nil, fmt.Errorf("no parent page URL passed")
	}

	var title string
	titleNode := doc.Find("title")
	l.Debugw("[collectLinks]", "titleNode", titleNode)
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
		l.Debugw("[collectLinks/loop]", "linkNode", linkNode, "linkHref", linkHref, "err", err)
		if err != nil {
			continue
		}
		links = append(links, orgLink{Link: linkUrl.String(), Title: linkNode.Text()})
	}

	return &title, links, nil
}

func perform(ctx *cli.Context) error {
	l := logger.Sugar()
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

	xkb.EnsureEnglishKeyboardLayout()
	sessionName, err := ui.GetSelection([]string{}, "save as", ctx.String(ui.SelectorToolFlagName), ctx.String(impl.SelectorFontFlagName), true, false)
	l.Debugw("[perform]", "sessionName", sessionName, "err", err)
	pageSoup, err := soup.Get(pageUrl.String())
	l.Debugw("[perform]", "pageSoup", pageSoup, "err", err)
	if err != nil {
		return err
	}

	title, links, err := collectLinks(soup.HTMLParse(pageSoup), pageUrl)
	if err != nil {
		return err
	}

	orgContent := []string{
		fmt.Sprintf("#+TITLE: %s", *title),
		fmt.Sprintf("#+PROPERTY: url %s", pageUrl.String()),
	}
	for _, link := range links {
		orgContent = append(orgContent, link.String())
	}

	orgFilename := fmt.Sprintf("%s/%s.org", ctx.String("export-path"), sessionName)
	l.Debugw("[perform]", "orgFilename", orgFilename)
	file, err := os.OpenFile(orgFilename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
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
			Name:     impl.SelectorFontFlagName,
			Aliases:  []string{"f"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_FONT"},
			Usage:    "Font to use for selector application, e.g. dmenu, rofi, etc.",
			Required: false,
		},
		&cli.StringFlag{
			Name:     ui.SelectorToolFlagName,
			Aliases:  []string{"T"},
			EnvVars:  []string{impl.EnvPrefix + "_SELECTOR_TOOL"},
			Value:    ui.SelectorToolDefault,
			Usage:    "Selector tool to use, e.g. dmenu, rofi, etc.",
			Required: false,
		},
	}
	app.Action = perform
	return app
}

func main() {
	logger = impl.NewLogger()
	defer logger.Sync()
	l := logger.Sugar()
	app := createCLI()
	err := app.Run(os.Args)
	if err != nil {
		l.Errorw("[main]", "err", err)
	}
}
