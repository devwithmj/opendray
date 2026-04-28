package catalog

import (
	"context"
	"fmt"

	"github.com/opendray/opendray-v2/internal/session"
)

// SessionProvider adapts Catalog to session.ProviderResolver. The
// session.Manager owns spawn-time scratch dirs; SessionProvider only
// supplies a Prepare callback that writes per-session MCP config into
// that scratch dir and contributes the args/env the provider's CLI
// needs to pick the file up.
type SessionProvider struct{ cat *Catalog }

func NewSessionProvider(cat *Catalog) *SessionProvider {
	return &SessionProvider{cat: cat}
}

func (sp *SessionProvider) Resolve(ctx context.Context, id string) (session.ProviderInfo, error) {
	p, err := sp.cat.Get(ctx, id)
	if err != nil {
		return session.ProviderInfo{}, err
	}
	if !p.Enabled {
		return session.ProviderInfo{}, fmt.Errorf("%w: %s is disabled", session.ErrProviderUnavailable, id)
	}

	exe := p.Manifest.Executable
	if v, ok := p.Config["command"].(string); ok && v != "" {
		exe = v
	}
	args := append([]string(nil), p.Manifest.DefaultArgs...)

	info := session.ProviderInfo{
		ID:         p.Manifest.ID,
		Executable: exe,
		Args:       args,
	}

	if !p.Manifest.Capabilities.SupportsMcp {
		return info, nil
	}
	servers := parseMCPServers(p.Config)
	if len(servers) == 0 {
		return info, nil
	}

	providerID := p.Manifest.ID
	info.Prepare = func(_ context.Context, _, baseDir string) (session.PrepareOutput, error) {
		extraArgs, env, err := renderMCP(providerID, baseDir, servers)
		if err != nil {
			return session.PrepareOutput{}, err
		}
		return session.PrepareOutput{Args: extraArgs, Env: env}, nil
	}
	return info, nil
}
