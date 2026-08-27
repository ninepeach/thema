package thema

import (
	"errors"
	"fmt"
)

var (
	ErrTemplateNotFound      = errors.New("thema: template not found")
	ErrInvalidTheme          = errors.New("thema: invalid theme")
	ErrInvalidPath           = errors.New("thema: invalid logical path")
	ErrIncompatibleTheme     = errors.New("thema: incompatible theme contract")
	ErrDuplicateContribution = errors.New("thema: duplicate contribution")
	ErrRender                = errors.New("thema: render failed")
	ErrRefresh               = errors.New("thema: refresh failed")
)

func invalidTheme(themeID, format string, args ...any) error {
	detail := fmt.Sprintf(format, args...)
	return fmt.Errorf("%w: theme %q: %s", ErrInvalidTheme, themeID, detail)
}

func incompatibleTheme(themeID, contract string) error {
	return fmt.Errorf("%w: theme %q requires contract %q; runtime supports %q", ErrIncompatibleTheme, themeID, contract, themeContractVersion)
}
