package ui

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"gioui.org/app"
	"git.sr.ht/~gioverse/chat/list"
)

// randomImage returns a random image at the given size.
// Downloads some number of random images from unplash and caches them on disk.
//
// TODO(jfm) [performance]: download images concurrently (parallel downloads,
// async to the gui event loop).
func randomImage(sz image.Point) (image.Image, error) {
	mkCacheDir := func(base string) string {
		return filepath.Join(base, "chat", fmt.Sprintf("%dx%d", sz.X, sz.Y))
	}
	cache := mkCacheDir(os.TempDir())
	if err := os.MkdirAll(cache, 0755); err != nil {
		if !errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("preparing cache directory: %w", err)
		}
		dir, err := app.DataDir()
		if err != nil {
			return nil, fmt.Errorf("failed finding application data dir: %w", err)
		}
		cache = mkCacheDir(dir)
		if err := os.MkdirAll(cache, 0755); err != nil {
			return nil, fmt.Errorf("preparing fallback cache directory: %w", err)
		}
	}
	entries, err := ioutil.ReadDir(cache)
	if err != nil {
		return nil, fmt.Errorf("reading cache entries: %w", err)
	}
	entries = filter(entries, isFile)
	if len(entries) == 0 {
		for ii := 0; ii < 10; ii++ {
			ii := ii
			if err := func() error {
				r, err := http.Get(fmt.Sprintf("https://source.unsplash.com/random/%dx%d?nature", sz.X, sz.Y))
				if err != nil {
					return fmt.Errorf("fetching image data: %w", err)
				}
				defer r.Body.Close()
				imgf, err := os.Create(filepath.Join(cache, strconv.Itoa(ii)))
				if err != nil {
					return fmt.Errorf("creating image file on disk: %w", err)
				}
				defer imgf.Close()
				if _, err := io.Copy(imgf, r.Body); err != nil {
					return fmt.Errorf("downloading image: %w", err)
				}
				return nil
			}(); err != nil {
				return nil, fmt.Errorf("populating image cache: %w", err)
			}
		}
		return randomImage(sz)
	}
	selection := entries[rand.Intn(len(entries))]
	imgf, err := os.Open(filepath.Join(cache, selection.Name()))
	if err != nil {
		return nil, fmt.Errorf("opening image file: %w", err)
	}
	defer imgf.Close()
	img, _, err := image.Decode(imgf)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	return img, nil
}

// isFile filters out non-file entries.
func isFile(info fs.FileInfo) bool {
	return !info.IsDir()
}

func filter(list []fs.FileInfo, predicate func(fs.FileInfo) bool) (filtered []fs.FileInfo) {
	for _, item := range list {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// dupSlice returns a slice composed of the same elements in the same order,
// but backed by a different array.
func dupSlice(in []list.Element) []list.Element {
	out := make([]list.Element, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

// sliceRemove takes the given index of a slice and swaps it with the final
// index in the slice, then shortens the slice by one element. This hides
// the element at index from the slice, though it does not erase its data.
func sliceRemove(s *[]list.Element, index int) {
	lastIndex := len(*s) - 1
	(*s)[index], (*s)[lastIndex] = (*s)[lastIndex], (*s)[index]
	*s = (*s)[:lastIndex]
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fetch image for the given id.
// Image is initially downloaded from the provided url and stored on disk.
func fetch(id, u string) (image.Image, error) {
	path := filepath.Join(os.TempDir(), "chat", "resources", id)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("preparing resource directory: %w", err)
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		if err := func() error {
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("creating resource file: %w", err)
			}
			defer f.Close()
			r, err := http.Get(u)
			if err != nil {
				return fmt.Errorf("GET: %w", err)
			}
			defer r.Body.Close()
			if r.StatusCode != http.StatusOK {
				return fmt.Errorf("GET: %s", r.Status)
			}
			if _, err := io.Copy(f, r.Body); err != nil {
				return fmt.Errorf("downloading resource to disk: %w", err)
			}
			return nil
		}(); err != nil {
			return nil, err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening resource file: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	return img, nil
}
