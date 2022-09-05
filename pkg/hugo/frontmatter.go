package hugo

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gohugoio/hugo/parser/pageparser"
)

const (
	fmTitle      = "title"
	fmAuthor     = "author"
	fmCategories = "categories"
	fmTags       = "tags"

	fmDate        = "date"        // priority high
	fmLastmod     = "lastmod"     // priority middle
	fmPublishDate = "publishDate" // priority low
)

type FrontMatter struct {
	Title    string
	Author   string
	Category string
	Tags     []string
	Date     time.Time
}

// ParseFrontMatter parses the frontmatter of the specified Hugo content.
func ParseFrontMatter(filename string, defaultFrontMatter *FrontMatter) (*FrontMatter, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseFrontMatter(file, defaultFrontMatter)
}

func parseFrontMatter(r io.Reader, defaultFrontMatter *FrontMatter) (*FrontMatter, error) {
	cfm, err := pageparser.ParseFrontMatterAndContent(r)
	if err != nil {
		return nil, err
	}

	fm := &FrontMatter{}
	if fm.Title, err = getString(&cfm, fmTitle); err != nil {
		fm.Title = defaultFrontMatter.Title
	}
	if isArray := isArray(&cfm, fmAuthor); isArray {
		if fm.Author, err = getFirstStringItem(&cfm, fmAuthor); err != nil {
			return nil, err
		}
	} else {
		if fm.Author, err = getString(&cfm, fmAuthor); err != nil {
			fm.Author = defaultFrontMatter.Author
		}
	}
	if fm.Category, err = getFirstStringItem(&cfm, fmCategories); err != nil {
		fm.Category = defaultFrontMatter.Category
	}
	if fm.Tags, err = getAllStringItems(&cfm, fmTags); err != nil {
		fm.Tags = defaultFrontMatter.Tags
	}
	if fm.Date, err = getContentDate(&cfm); err != nil {
		fm.Date = defaultFrontMatter.Date
	}

	if isDraft, err := getBool(&cfm, "draft"); err != nil && isDraft {
		return nil, errors.New("this article is draft")
	}

	return fm, nil
}

func getContentDate(cfm *pageparser.ContentFrontMatter) (time.Time, error) {
	for _, key := range []string{fmDate, fmLastmod, fmPublishDate} {
		t, err := getTime(cfm, key)
		if err != nil {
			switch err.(type) {
			case *FMNotExistError:
				continue
			}
		}
		return t, err
	}
	return time.Now(), NewFMNotExistError(
		strings.Join([]string{fmDate, fmLastmod, fmPublishDate}, ", "))
}

func getTime(cfm *pageparser.ContentFrontMatter, fmKey string) (time.Time, error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return time.Now(), NewFMNotExistError(fmKey)
	}
	switch t := v.(type) {
	case string:
		return time.Parse(time.RFC3339, t)
	case time.Time:
		return t, nil
	default:
		return time.Now(), NewFMInvalidTypeError(fmKey, "time.Time or string", t)
	}
}

func getString(cfm *pageparser.ContentFrontMatter, fmKey string) (string, error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return "", NewFMNotExistError(fmKey)
	}

	switch s := v.(type) {
	case string:
		if s == "" {
			return "", NewFMNotExistError(fmKey)
		}
		return s, nil
	default:
		return "", NewFMInvalidTypeError(fmKey, "string", s)
	}
}

func getAllStringItems(cfm *pageparser.ContentFrontMatter, fmKey string) ([]string, error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return nil, NewFMNotExistError(fmKey)
	}

	switch arr := v.(type) {
	case []interface{}:
		var strarr []string
		for _, item := range arr {
			switch s := item.(type) {
			case string:
				if s != "" {
					strarr = append(strarr, s)
				}
			default:
				return nil, NewFMInvalidTypeError(fmKey, "string", s)
			}
		}
		if len(strarr) < 1 {
			return nil, NewFMNotExistError(fmKey)
		}
		return strarr, nil

	default:
		return nil, NewFMInvalidTypeError(fmKey, "[]interface{}", arr)
	}
}

func getFirstStringItem(cfm *pageparser.ContentFrontMatter, fmKey string) (string, error) {
	arr, err := getAllStringItems(cfm, fmKey)
	if err != nil {
		return "", err
	}
	return arr[0], nil
}

func getBool(cfm *pageparser.ContentFrontMatter, fmKey string) (bool, error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return false, NewFMNotExistError(fmKey)
	}

	switch b := v.(type) {
	case bool:
		return b, nil
	default:
		return false, NewFMInvalidTypeError(fmKey, "bool", b)
	}
}

func isArray(cfm *pageparser.ContentFrontMatter, fmKey string) bool {
	switch v := cfm.FrontMatter[fmKey]; v.(type) {
	case []interface{}:
		return true
	default:
		return false
	}
}
