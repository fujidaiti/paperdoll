package newspaper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"
)

// EditorialInterval is a segment between consecutive two schedule datetimes.
type EditorialInterval struct {
	Next time.Time
	Last time.Time
}

func (ei *EditorialInterval) Valid() bool {
	return !ei.Next.IsZero() && !ei.Last.IsZero() && ei.Next.After(ei.Last)
}

// Contains reports whether t falls within the interval.
// Always returns false if Valid also reports false.
func (ei *EditorialInterval) Contains(t time.Time) bool {
	return ei.Valid() && t.After(ei.Last) && t.Before(ei.Next)
}

// TODO(#154): rename Schedule to PublicationSchedule
func getSchedules(ctx context.Context, db *sql.DB, t time.Time) ([]time.Time, error) {
	rows, err := db.QueryContext(ctx, `SELECT minute_of_date FROM newspaper_schedules;`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	year, month, day := t.Date()
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	var schedules []time.Time
	for rows.Next() {
		var dt int
		err := rows.Scan(&dt)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, midnight.Add(time.Duration(dt)*time.Minute))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}

func FindEditorialInterval(
	ctx context.Context, db *sql.DB, t time.Time,
) (EditorialInterval, error) {
	ss, err := getSchedules(ctx, db, t)
	if err != nil {
		return EditorialInterval{}, err
	}
	if len(ss) == 0 {
		return EditorialInterval{}, errors.New("no schedule is registered")
	}
	slices.SortFunc(ss, time.Time.Compare)
	return findEditorialInterval(t, ss), nil
}

// findEditorialInterval returns the nearest consecutive pair of datetime points
// surrounding the given time t on the given daily publishing schedule timeline ss.
// The returned datetimes may be from yesterday or tomorrow if the t falls before
// the first or after the last datetime in the timeline.
//
// If a datetime in ss equals t, it is returned as the left (last) point.
//
// The day of t and the Time instances in ss must be the same, and ss must
// be sorted in ascending order.
func findEditorialInterval(t time.Time, ss []time.Time) EditorialInterval {
	idx := -1
	for i, s := range ss {
		if s.After(t) {
			idx = i
			break
		}
	}

	var ei EditorialInterval
	switch {
	case idx == 0:
		// Yesterday's last schedule
		ei.Last = ss[len(ss)-1].AddDate(0, 0, -1)
		ei.Next = ss[0]

	case idx-1 >= 0:
		ei.Last = ss[idx-1]
		ei.Next = ss[idx]

	case idx < 0:
		ei.Last = ss[len(ss)-1]
		// Tomorrow's first schedule
		ei.Next = ss[0].AddDate(0, 0, 1)
	}
	return ei
}
