package ticketbot

import (
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

func intToPgInt4(i int, zeroIsNull bool) pgtype.Int4 {
	valid := true
	if zeroIsNull && i == 0 {
		valid = false
	}
	return pgtype.Int4{
		Int32: int32(i),
		Valid: valid,
	}
}

func stringToPgText(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  s != "",
	}
}

func timeToPgTimeStamp(t time.Time, zeroIsNull bool) pgtype.Timestamp {
	valid := true
	if zeroIsNull && t.IsZero() {
		valid = false
	}

	return pgtype.Timestamp{
		Time:             t,
		InfinityModifier: pgtype.Finite,
		Valid:            valid,
	}
}
