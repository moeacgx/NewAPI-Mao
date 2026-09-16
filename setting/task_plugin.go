package setting

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	TaskPluginEnabledKey            = "TaskPluginEnabled"
	TaskPluginMarketplaceSourcesKey = "TaskPluginMarketplaceSources"
)

var DefaultTaskPluginMarketplaceSources = []TaskPluginMarketplaceSource{
	{Name: "newapi", IndexURL: "https://www.newapi.ai/api/v1/plugins/index.json"},
	{Name: "maolaonewapi-plugins", IndexURL: "https://raw.githubusercontent.com/moeacgx/maolaonewapi-plugins/main/index.json"},
}

type TaskPluginMarketplaceSource struct {
	Name     string `json:"name"`
	IndexURL string `json:"index_url"`
}

func ValidateTaskPluginMarketplaceSources(sources []TaskPluginMarketplaceSource) error {
	if len(sources) > 16 {
		return errors.New("marketplace sources must contain at most 16 items")
	}
	seen := make(map[string]struct{}, len(sources))
	seenURLs := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Name) != source.Name {
			return errors.New("marketplace source name must not be empty")
		}
		if utf8.RuneCountInString(source.Name) > 128 {
			return errors.New("marketplace source name is too long")
		}
		if _, ok := seen[source.Name]; ok {
			return fmt.Errorf("duplicate marketplace source name: %s", source.Name)
		}
		seen[source.Name] = struct{}{}
		if _, ok := seenURLs[source.IndexURL]; ok {
			return fmt.Errorf("duplicate marketplace source URL: %s", source.IndexURL)
		}
		seenURLs[source.IndexURL] = struct{}{}
		if len(source.IndexURL) > 2048 {
			return errors.New("marketplace source URL is too long")
		}
		parsed, err := url.Parse(source.IndexURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.ContainsAny(source.IndexURL, "#\\") {
			return errors.New("marketplace source URL must be an https URL without credentials, query, or fragment")
		}
	}
	return nil
}
