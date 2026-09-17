package cwsvc

import (
	"context"
	"fmt"
	"sort"

	"github.com/thecoretg/ticketbot/models"
)

// ListPriorities fetches the ConnectWise priority list live. Priorities are not stored locally:
// there are only a handful and they rarely change.
func (s *Service) ListPriorities(ctx context.Context) ([]*models.Priority, error) {
	type cwPriority struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level string `json:"level,omitempty"`
		Sort  int    `json:"sort,omitempty"`
	}

	items, err := s.CWClient.GetMany[cwPriority](ctx, "service/priorities", map[string]string{"fields": "id,name,level,sort"})
	if err != nil {
		return nil, fmt.Errorf("listing priorities: %w", err)
	}

	out := make([]*models.Priority, 0, len(items))
	for _, p := range items {
		out = append(out, &models.Priority{ID: p.ID, Name: p.Name, Level: p.Level, Sort: p.Sort})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Sort != out[j].Sort {
			return out[i].Sort < out[j].Sort
		}
		return out[i].Name < out[j].Name
	})

	return out, nil
}
