package cmd

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

const (
	newsSearchEndpoint  = "quote.news_search"
	newsKeywordMaxChars = 200
	newsSourceMaxChars  = 200
	newsPageSizeMin     = 1
	newsPageSizeMax     = 50
	newsTypeAll         = "ALL"
	newsTypePost        = "POST"
	newsTypeNotice      = "NOTICE"
	newsTypeReport      = "REPORT"
	newsSortTime        = "TIME"
	newsSortPV          = "PV"
	newsLangSimplified  = "zh-CN"
	newsLangTraditional = "zh-HK"
	newsLangEnglish     = "en"
	newsLangJapanese    = "ja"
	newsSourceIDFormat  = `^[0-9]+(,[0-9]+)*$`
)

var newsSourceIDsPattern = regexp.MustCompile(newsSourceIDFormat)

func validateNewsSearch(cmd *cobra.Command, args []string, query url.Values) error {
	if len(args) == 0 {
		keyword, err := cmd.Flags().GetString("keyword")
		if err != nil || len(strings.TrimSpace(keyword)) == 0 {
			return fmt.Errorf("news search keyword is required")
		}
	}
	keyword := strings.TrimSpace(query.Get("keyword"))
	if len(keyword) == 0 || utf8.RuneCountInString(keyword) > newsKeywordMaxChars {
		return fmt.Errorf("news search keyword must contain 1–%d characters", newsKeywordMaxChars)
	}
	if err := validateNewsFilters(query); err != nil {
		return err
	}
	return validateNewsPageSize(query.Get("page_size"))
}

func validateNewsFilters(query url.Values) error {
	if !newsValueIn(query.Get("news_type"), newsTypeAll, newsTypePost, newsTypeNotice, newsTypeReport) {
		return fmt.Errorf("invalid --news-type: use ALL, POST, NOTICE or REPORT")
	}
	if !newsValueIn(query.Get("sort_type"), newsSortTime, newsSortPV) {
		return fmt.Errorf("invalid --sort-type for news: use TIME or PV")
	}
	lang := query.Get("lang")
	if len(lang) > 0 && !newsValueIn(lang, newsLangSimplified, newsLangTraditional, newsLangEnglish, newsLangJapanese) {
		return fmt.Errorf("invalid --lang: use zh-CN, zh-HK, en or ja")
	}
	for _, key := range []string{"source_ids", "sources"} {
		if utf8.RuneCountInString(query.Get(key)) > newsSourceMaxChars {
			return fmt.Errorf("%s must not exceed %d characters", key, newsSourceMaxChars)
		}
	}
	ids := query.Get("source_ids")
	if len(ids) > 0 && !newsSourceIDsPattern.MatchString(ids) {
		return fmt.Errorf("--source-ids must be comma-separated numeric IDs")
	}
	return nil
}

func newsValueIn(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func validateNewsPageSize(value string) error {
	pageSize, err := strconv.Atoi(value)
	if err != nil || pageSize < newsPageSizeMin || pageSize > newsPageSizeMax {
		return fmt.Errorf("--page-size must be between %d and %d", newsPageSizeMin, newsPageSizeMax)
	}
	return nil
}
