package http

import (
	"fmt"
	"net/url"
	"strings"
)

func validateAd(title, photoURL string) error {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 200 {
		return fmt.Errorf("title must contain between 1 and 200 characters")
	}
	if len(photoURL) > 2048 {
		return fmt.Errorf("photo_url must not exceed 2048 characters")
	}
	u, err := url.ParseRequestURI(photoURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return fmt.Errorf("photo_url must be an absolute HTTP or HTTPS URL")
	}
	return nil
}
