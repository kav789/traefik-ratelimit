package traefik_ratelimit

import (
	"fmt"
	"os"
)

func (g *GlobalRateLimit) setFromFile() error {
	g.umtx.Lock()
	defer g.umtx.Unlock()
	if g.config == nil {
		return fmt.Errorf("config not specified")
	}
	b, err := os.ReadFile(g.config.RatelimitPath)
	if err != nil {
		return err
	}
	return g.update(b)
}

func (g *GlobalRateLimit) setFromSettings() error {
	g.umtx.Lock()
	defer g.umtx.Unlock()
	if g.config == nil {
		return fmt.Errorf("config not specified")
	}
	result, err := g.settings.Get(g.config.KeeperRateLimitKey)
	if err != nil {
		return err
	}

	if result != nil && !g.version.Equal(result) {
		if g.version != nil {
			locallog("old configuration: ", g.version.Version, g.version.ModRevision)
		}
		err = g.update([]byte(result.Value))
		if err != nil {
			return err
		}
		g.version = result
		locallog("new configuration loaded: ", g.version.Version, g.version.ModRevision)
	}
	return nil
}

func (r *RateLimit) Update(b []byte) error {
	return grl.update(b)
}
