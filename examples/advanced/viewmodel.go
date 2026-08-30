package main

type SiteView struct {
	Name string
}

type PageMeta struct {
	Title       string
	Description string
}

type Service struct {
	Title       string
	Description string
}

type Project struct {
	Name        string
	Category    string
	Description string
	Result      string
	Image       string
}

type Stat struct {
	Value string
	Label string
}

type PageView struct {
	Site        SiteView
	Page        PageMeta
	CurrentPath string
	Locale      string
	Services    []Service
	Projects    []Project
	Stats       []Stat
}

func pageView(path, locale, title, description string) PageView {
	return PageView{
		Site:        SiteView{Name: "Northstar"},
		Page:        PageMeta{Title: title, Description: description},
		CurrentPath: path,
		Locale:      locale,
	}
}

func serviceViews() []Service {
	return []Service{
		{Title: "services.product.title", Description: "services.product.description"},
		{Title: "services.cloud.title", Description: "services.cloud.description"},
		{Title: "services.modernization.title", Description: "services.modernization.description"},
	}
}

func projectViews() []Project {
	return []Project{
		{
			Name:        "Atlas",
			Category:    "work.atlas.category",
			Description: "work.atlas.description",
			Result:      "work.atlas.result",
			Image:       "images/project-atlas.svg",
		},
		{
			Name:        "Orbit",
			Category:    "work.orbit.category",
			Description: "work.orbit.description",
			Result:      "work.orbit.result",
			Image:       "images/project-orbit.svg",
		},
	}
}

func statViews() []Stat {
	return []Stat{
		{Value: "120+", Label: "stats.projects"},
		{Value: "30+", Label: "stats.clients"},
		{Value: "8", Label: "stats.years"},
	}
}
