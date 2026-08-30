package main

import (
	"net/url"
	"strings"
)

type SiteView struct {
	Name string
}

type PageMeta struct {
	Title       string
	Description string
}

type LanguageOption struct {
	Code    string
	Name    string
	URL     string
	Current bool
}

type BreadcrumbItem struct {
	Label   string
	URL     string
	Current bool
}

type PageView struct {
	Site        SiteView
	Page        PageMeta
	ActiveNav   string
	Locale      string
	LocaleQuery string
	Language    LanguageOption
	Languages   []LanguageOption
	Breadcrumbs []BreadcrumbItem
}

func pageView(path, activeNav, locale, title, description string) PageView {
	query := localeQuery(locale)
	return PageView{
		Site:        SiteView{Name: "KHotel"},
		Page:        PageMeta{Title: title, Description: description},
		ActiveNav:   activeNav,
		Locale:      locale,
		LocaleQuery: query,
		Language:    currentLanguage(locale),
		Languages:   languageOptions(path, locale),
		Breadcrumbs: breadcrumbs(path, locale, title),
	}
}

func localeQuery(locale string) string {
	if locale == "en" {
		return ""
	}
	return "?" + url.Values{"lang": []string{locale}}.Encode()
}

func currentLanguage(locale string) LanguageOption {
	for _, language := range supportedLanguages() {
		if language.Code == locale {
			language.Current = true
			return language
		}
	}
	return supportedLanguages()[0]
}

func languageOptions(path, locale string) []LanguageOption {
	languages := supportedLanguages()
	for index := range languages {
		languages[index].URL = path + "?" + url.Values{"lang": []string{languages[index].Code}}.Encode()
		languages[index].Current = languages[index].Code == locale
	}
	return languages
}

func supportedLanguages() []LanguageOption {
	return []LanguageOption{
		{Code: "en", Name: "English"},
		{Code: "ja", Name: "日本語"},
		{Code: "zh-Hant", Name: "繁體中文"},
		{Code: "zh-Hans", Name: "简体中文"},
	}
}

func breadcrumbs(path, locale, title string) []BreadcrumbItem {
	if path == "/" {
		return nil
	}
	query := localeQuery(locale)
	items := []BreadcrumbItem{{Label: "nav.home", URL: "/" + query}}
	if strings.HasPrefix(path, "/rooms/") {
		items = append(items, BreadcrumbItem{Label: "nav.rooms", URL: "/rooms" + query})
	}
	return append(items, BreadcrumbItem{Label: title, Current: true})
}
