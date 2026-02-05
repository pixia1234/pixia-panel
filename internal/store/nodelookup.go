package store

import "context"

// LookupBySecret implements gost.NodeLookup.
func (s *Store) LookupBySecret(ctx context.Context, secret string) (int64, error) {
	node, err := s.GetNodeBySecret(ctx, secret)
	if err != nil {
		return 0, err
	}
	return node.ID, nil
}
