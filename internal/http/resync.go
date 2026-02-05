package httpapi

import (
	"context"

	"pixia-panel/internal/store"
)

// ResyncNode pushes limiters and services for a node when it reconnects.
func (s *Server) ResyncNode(ctx context.Context, nodeID int64) {
	tunnels, err := s.store.ListTunnels(ctx)
	if err != nil {
		return
	}

	tunnelMap := make(map[int64]store.Tunnel, len(tunnels))
	for _, t := range tunnels {
		tunnelMap[t.ID] = t
		if t.InNodeID == nodeID {
			limits, err := s.store.ListActiveSpeedLimitsByTunnel(ctx, t.ID)
			if err != nil {
				continue
			}
			for i := range limits {
				limiterID := limits[i].ID
				s.ensureLimiterConfig(ctx, t.InNodeID, &limiterID)
			}
		}
	}

	forwards, err := s.store.ListForwardsAll(ctx)
	if err != nil {
		return
	}

	for i := range forwards {
		fwItem := forwards[i]
		tunnel, ok := tunnelMap[fwItem.TunnelID]
		if !ok {
			continue
		}
		if tunnel.InNodeID != nodeID && !(tunnel.Type == 2 && tunnel.OutNodeID == nodeID) {
			continue
		}
		limiter := s.resolveSpeedLimiterCtx(ctx, fwItem.UserID, fwItem.TunnelID)
		fw := fwItem.Forward
		s.enqueueForwardGostCtx(ctx, &fw, &tunnel, limiter, "UpdateService")
	}
}
