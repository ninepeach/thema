package main

type SiteView struct {
	Name string
}

type PageMeta struct {
	Title       string
	Description string
}

type PageView struct {
	Site        SiteView
	Page        PageMeta
	CurrentPath string
	ActiveNav   string
	Locale      string
}

func pageView(path, activeNav, locale, title, description string) PageView {
	return PageView{
		Site:        SiteView{Name: "KHotel"},
		Page:        PageMeta{Title: title, Description: description},
		CurrentPath: path,
		ActiveNav:   activeNav,
		Locale:      locale,
	}
}
