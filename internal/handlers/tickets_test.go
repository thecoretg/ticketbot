package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/thecoretg/ticketbot/models"
)

func TestTicketFilterFromQuery(t *testing.T) {
	tests := []struct {
		query    string
		wantErr  bool
		wantSort models.TicketSort
		wantDesc bool
	}{
		{query: "", wantSort: models.TicketSortUpdated, wantDesc: true},
		{query: "sort=updated", wantSort: models.TicketSortUpdated, wantDesc: true},
		{query: "sort=updated&dir=asc", wantSort: models.TicketSortUpdated, wantDesc: false},
		{query: "sort=summary", wantSort: models.TicketSortSummary, wantDesc: false},
		{query: "sort=company&dir=desc", wantSort: models.TicketSortCompany, wantDesc: true},
		{query: "sort=id&dir=asc", wantSort: models.TicketSortID},
		{query: "dir=asc", wantSort: models.TicketSortUpdated, wantDesc: false},
		{query: "sort=raw", wantErr: true},
		{query: "sort=updated_on", wantErr: true},
		{query: "sort=summary%20desc", wantErr: true},
		{query: "sort=summary&dir=up", wantErr: true},
		{query: "sort=SUMMARY", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			f, err := ticketFilterFromQuery(httptest.NewRequest("GET", "/tickets?"+tt.query, nil))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %+v", f)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if f.Sort != tt.wantSort || f.SortDesc != tt.wantDesc {
				t.Errorf("sort=%q desc=%v, want %q %v", f.Sort, f.SortDesc, tt.wantSort, tt.wantDesc)
			}
		})
	}
}

func TestTicketFilterFromQueryKeepsFilters(t *testing.T) {
	f, err := ticketFilterFromQuery(httptest.NewRequest("GET",
		"/tickets?board_id=3&status_id=7&closed=false&q=vpn&sort=owner&dir=desc&page=2&page_size=25", nil))
	if err != nil {
		t.Fatal(err)
	}
	if f.BoardID == nil || *f.BoardID != 3 || f.StatusID == nil || *f.StatusID != 7 ||
		f.Closed == nil || *f.Closed || f.Search != "vpn" || f.Page != 2 || f.PageSize != 25 ||
		f.Sort != models.TicketSortOwner || !f.SortDesc {
		t.Errorf("filter = %+v", f)
	}

	if _, err := ticketFilterFromQuery(httptest.NewRequest("GET", "/tickets?page=x", nil)); err == nil {
		t.Error("bad page: want an error")
	}
}
