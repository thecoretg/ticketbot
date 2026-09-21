// Package transfer exports workflows to a portable bundle and imports one, creating the Webex
// recipients and lists it needs and rewriting the references. It is how workflows built on one
// instance move to another without being rebuilt by hand.
package transfer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/lists"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

const (
	ResultCreated  = "created"
	ResultReplaced = "replaced"
	ResultSkipped  = "skipped"
	ResultError    = "error"
)

type Service struct {
	Workflows  *workflow.Service
	Lists      *lists.Service
	Recipients repos.WebexRecipientRepository
	Boards     repos.BoardRepository
	now        func() time.Time
}

type Params struct {
	Workflows  *workflow.Service
	Lists      *lists.Service
	Recipients repos.WebexRecipientRepository
	Boards     repos.BoardRepository
}

func New(p Params) *Service {
	return &Service{Workflows: p.Workflows, Lists: p.Lists, Recipients: p.Recipients, Boards: p.Boards, now: time.Now}
}

// Export bundles the given workflows, or every workflow when ids is empty.
func (s *Service) Export(ctx context.Context, ids []int) (*models.WorkflowBundle, error) {
	var wfs []*models.Workflow
	if len(ids) == 0 {
		all, err := s.Workflows.List(ctx)
		if err != nil {
			return nil, err
		}
		wfs = all
	} else {
		for _, id := range ids {
			w, err := s.Workflows.Get(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("workflow %d: %w", id, err)
			}
			wfs = append(wfs, w)
		}
	}

	b := &models.WorkflowBundle{
		Version:    models.WorkflowBundleVersion,
		ExportedAt: s.now().UTC(),
		Recipients: []models.BundleRecipient{},
		Lists:      []models.BundleList{},
		Workflows:  make([]models.Workflow, 0, len(wfs)),
	}
	seenRecip, seenList := map[int]bool{}, map[int]bool{}

	for _, w := range wfs {
		cp := *w
		cp.ID, cp.CreatedOn, cp.UpdatedOn = 0, time.Time{}, time.Time{}
		b.Workflows = append(b.Workflows, cp)

		for _, n := range w.Nodes {
			if n.Notify != nil && n.Notify.RecipientID != nil && !seenRecip[*n.Notify.RecipientID] {
				seenRecip[*n.Notify.RecipientID] = true
				r, err := s.Recipients.Get(ctx, *n.Notify.RecipientID)
				if err != nil {
					return nil, fmt.Errorf("recipient %d used by %q: %w", *n.Notify.RecipientID, n.Title, err)
				}
				b.Recipients = append(b.Recipients, models.BundleRecipient{ID: r.ID, WebexID: r.WebexID, Name: r.Name, Email: r.Email, Type: r.Type})
			}
			for _, id := range listIDs(n.Condition) {
				if seenList[id] {
					continue
				}
				seenList[id] = true
				d, err := s.Lists.Get(ctx, id)
				if err != nil {
					return nil, fmt.Errorf("list %d used by %q: %w", id, n.Title, err)
				}
				bl := models.BundleList{ID: d.ID, Name: d.Name, ItemType: d.ItemType, Description: d.Description, Items: []int{}}
				for _, it := range d.Items {
					bl.Items = append(bl.Items, it.ItemID)
				}
				b.Lists = append(b.Lists, bl)
			}
		}
	}
	return b, nil
}

// Import creates what the bundle needs and then each workflow. Nothing is transactional across
// workflows: the report says what happened to each one, and a re-run is safe because recipients
// and lists are matched before they are created.
func (s *Service) Import(ctx context.Context, b *models.WorkflowBundle, opts models.ImportOptions) (*models.ImportReport, error) {
	if b == nil {
		return nil, errors.New("empty bundle")
	}
	if b.Version != models.WorkflowBundleVersion {
		return nil, fmt.Errorf("unsupported bundle version %d (this build reads %d)", b.Version, models.WorkflowBundleVersion)
	}
	rep := &models.ImportReport{RecipientsCreated: []string{}, ListsCreated: []string{}, Workflows: []models.ImportWorkflowItem{}, Warnings: []string{}}

	recipMap := map[int]int{}
	for _, r := range b.Recipients {
		id, created, err := s.ensureRecipient(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("recipient %q: %w", r.Name, err)
		}
		recipMap[r.ID] = id
		if created {
			rep.RecipientsCreated = append(rep.RecipientsCreated, r.Name)
		}
	}

	listMap := map[int]int{}
	for _, l := range b.Lists {
		id, created, warnings, err := s.ensureList(ctx, l)
		if err != nil {
			return nil, fmt.Errorf("list %q: %w", l.Name, err)
		}
		listMap[l.ID] = id
		rep.Warnings = append(rep.Warnings, warnings...)
		if created {
			rep.ListsCreated = append(rep.ListsCreated, l.Name)
		}
	}

	for i := range b.Workflows {
		w := b.Workflows[i]
		item := models.ImportWorkflowItem{Name: w.Name, BoardID: w.BoardID, BoardName: w.BoardName}
		if err := rewrite(&w, recipMap, listMap); err != nil {
			item.Result, item.Error = ResultError, err.Error()
			rep.Workflows = append(rep.Workflows, item)
			continue
		}
		w.ID = 0

		if _, err := s.Boards.Get(ctx, w.BoardID); err != nil {
			item.Result, item.Error = ResultError, fmt.Sprintf("board %d is not synced on this instance; run a board sync first", w.BoardID)
			rep.Workflows = append(rep.Workflows, item)
			continue
		}

		existing, err := s.Workflows.GetByBoard(ctx, w.BoardID)
		switch {
		case err == nil && !opts.Replace:
			item.Result, item.ID = ResultSkipped, existing.ID
			item.Error = "this board already has a workflow; import again with replace to overwrite it"
		case err == nil:
			saved, rerr := s.Workflows.Replace(ctx, existing.ID, &w)
			if rerr != nil {
				item.Result, item.Error = ResultError, rerr.Error()
			} else {
				item.Result, item.ID = ResultReplaced, saved.ID
			}
		case errors.Is(err, models.ErrWorkflowNotFound):
			saved, cerr := s.Workflows.Create(ctx, &w)
			if cerr != nil {
				item.Result, item.Error = ResultError, cerr.Error()
			} else {
				item.Result, item.ID = ResultCreated, saved.ID
			}
		default:
			item.Result, item.Error = ResultError, err.Error()
		}
		rep.Workflows = append(rep.Workflows, item)
	}
	return rep, nil
}

// ensureRecipient finds the recipient by Webex id (stable across instances of one Webex org)
// and creates it when absent.
func (s *Service) ensureRecipient(ctx context.Context, r models.BundleRecipient) (int, bool, error) {
	if r.WebexID == "" {
		return 0, false, errors.New("bundle recipient has no webex_id")
	}
	if got, err := s.Recipients.GetByWebexID(ctx, r.WebexID); err == nil {
		return got.ID, false, nil
	} else if !errors.Is(err, models.ErrWebexRecipientNotFound) {
		return 0, false, err
	}
	created, err := s.Recipients.Upsert(ctx, &models.WebexRecipient{WebexID: r.WebexID, Name: r.Name, Email: r.Email, Type: r.Type})
	if err != nil {
		return 0, false, err
	}
	return created.ID, true, nil
}

// ensureList matches a list by name and item type and creates it with its members when absent.
// Members the synced ConnectWise data does not know are skipped with a warning.
func (s *Service) ensureList(ctx context.Context, l models.BundleList) (int, bool, []string, error) {
	all, err := s.Lists.List(ctx)
	if err != nil {
		return 0, false, nil, err
	}
	for _, e := range all {
		if strings.EqualFold(e.Name, l.Name) {
			if e.ItemType != l.ItemType {
				return 0, false, nil, fmt.Errorf("exists here with item type %s, bundle has %s", e.ItemType, l.ItemType)
			}
			return e.ID, false, nil, nil
		}
	}

	created, err := s.Lists.Create(ctx, &models.List{Name: l.Name, ItemType: l.ItemType, Description: l.Description})
	if err != nil {
		return 0, false, nil, err
	}
	var warnings []string
	for _, item := range l.Items {
		if _, err := s.Lists.AddItem(ctx, created.ID, item); err != nil {
			warnings = append(warnings, fmt.Sprintf("list %q: item %d skipped: %v", l.Name, item, err))
		}
	}
	return created.ID, true, warnings, nil
}

// rewrite points notify nodes and `in list N` conditions at this instance's ids.
func rewrite(w *models.Workflow, recipMap, listMap map[int]int) error {
	for i := range w.Nodes {
		n := &w.Nodes[i]
		if n.Notify != nil && n.Notify.RecipientID != nil {
			id, ok := recipMap[*n.Notify.RecipientID]
			if !ok {
				return fmt.Errorf("node %q names recipient %d, which the bundle does not carry", n.Title, *n.Notify.RecipientID)
			}
			n.Notify.RecipientID = &id
		}
		if n.Condition != "" {
			cond, err := rewriteLists(n.Condition, listMap)
			if err != nil {
				return fmt.Errorf("node %q: %w", n.Title, err)
			}
			n.Condition = cond
		}
	}
	return nil
}

// rewriteLists replaces each list id token in a condition, working from the end so earlier
// byte offsets stay valid.
func rewriteLists(cond string, listMap map[int]int) (string, error) {
	q, err := cwquery.Compile(cond)
	if err != nil {
		return "", fmt.Errorf("condition does not compile: %w", err)
	}
	refs := cwquery.ListRefs(q.Expr)
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		to, ok := listMap[ref.ListID]
		if !ok {
			return "", fmt.Errorf("condition references list %d, which the bundle does not carry", ref.ListID)
		}
		old := strconv.Itoa(ref.ListID)
		if ref.Pos < 0 || ref.Pos+len(old) > len(cond) || cond[ref.Pos:ref.Pos+len(old)] != old {
			return "", fmt.Errorf("could not locate list %d in condition", ref.ListID)
		}
		cond = cond[:ref.Pos] + strconv.Itoa(to) + cond[ref.Pos+len(old):]
	}
	return cond, nil
}

// listIDs returns the list ids a condition references; an uncompilable condition has none.
func listIDs(cond string) []int {
	if strings.TrimSpace(cond) == "" {
		return nil
	}
	q, err := cwquery.Compile(cond)
	if err != nil {
		return nil
	}
	var out []int
	for _, r := range cwquery.ListRefs(q.Expr) {
		out = append(out, r.ListID)
	}
	return out
}
