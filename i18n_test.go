package thema

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestBuiltInTranslatorFallbackInterpolationAndEscaping(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `<h1>{{t "home.title"}}</h1><p>{{t "guest.hello" "name" .Name}}</p>`,
	}, map[string]string{
		"en": `{"home":{"title":"Welcome"},"guest.hello":"Hello, {{name}}!"}`,
		"ja": `{"home.title":"ようこそ","guest":{"hello":"こんにちは、{{name}}！"}}`,
		"zh": `{"home.title":"欢迎"}`,
	}, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	assertRender(t, views, "pages/home", map[string]string{"Name": `<Stone>`}, `<h1>ようこそ</h1><p>こんにちは、&lt;Stone&gt;！</p>`, WithLocale("ja"))
	assertRender(t, views, "pages/home", map[string]string{"Name": "Stone"}, `<h1>欢迎</h1><p>Hello, Stone!</p>`, WithLocale("zh-CN"))
	assertRender(t, views, "pages/home", map[string]string{"Name": "Stone"}, `<h1>Welcome</h1><p>Hello, Stone!</p>`, WithLocale("fr"))
}

func TestConcurrentPerRenderLocalesDoNotLeak(t *testing.T) {
	repository := newTestTheme(t, map[string]string{
		"pages/home.html": `{{t "message"}}`,
	}, map[string]string{
		"en": `{"message":"English"}`,
		"ja": `{"message":"日本語"}`,
	}, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	const renders = 100
	var wait sync.WaitGroup
	errorsFound := make(chan error, renders)
	for i := 0; i < renders; i++ {
		locale, want := "en", "English"
		if i%2 == 1 {
			locale, want = "ja", "日本語"
		}
		wait.Add(1)
		go func() {
			defer wait.Done()
			var output bytes.Buffer
			if err := views.Render(context.Background(), &output, "pages/home", nil, WithLocale(locale)); err != nil {
				errorsFound <- err
				return
			}
			if output.String() != want {
				errorsFound <- fmt.Errorf("locale %s rendered %q, want %q", locale, output.String(), want)
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func TestMissingTranslationReturnsKey(t *testing.T) {
	repository := newTestTheme(t, map[string]string{"pages/home.html": `{{t "missing.key"}}`}, nil, nil)
	views, err := New(repository, "default")
	if err != nil {
		t.Fatal(err)
	}
	assertRender(t, views, "pages/home", nil, "missing.key")
}

