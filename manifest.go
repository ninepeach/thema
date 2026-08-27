package thema

import "strings"

const themeContractVersion = "0.1"

type manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Thema   string `json:"thema"`
}

func (m manifest) validate(themeID string) error {
	if strings.TrimSpace(m.Name) == "" || m.Version == "" || m.Thema == "" {
		return invalidTheme(themeID, "theme.json requires name, version, and thema")
	}
	if !validSemver(m.Version) {
		return invalidTheme(themeID, "theme version %q is not valid SemVer", m.Version)
	}
	if m.Thema != themeContractVersion {
		return incompatibleTheme(themeID, m.Thema)
	}
	return nil
}
