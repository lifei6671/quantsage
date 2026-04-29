package browserfetch

import (
	"fmt"
	"strings"

	fakeuseragent "github.com/lib4u/fake-useragent"
)

type userAgentResolver func(normalizedConfig) (string, error)

func defaultUserAgentResolver(cfg normalizedConfig) (string, error) {
	generator, err := fakeuseragent.New()
	if err != nil {
		return "", fmt.Errorf("init fake user agent generator: %w", err)
	}

	filter := generator.Filter().Chrome()
	switch cfg.UserAgentPlatform {
	case UserAgentPlatformMobile:
		filter.Platform(fakeuseragent.Mobile)
	default:
		filter.Platform(fakeuseragent.Desktop)
	}

	userAgent := strings.TrimSpace(filter.Get())
	if userAgent == "" {
		return "", fmt.Errorf("generate fake user agent: empty result")
	}

	return userAgent, nil
}
