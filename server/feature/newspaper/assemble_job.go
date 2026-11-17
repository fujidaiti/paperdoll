package newspaper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fujidaiti/paperdoll/server/feature/user"
)

// TODO(#140): do the same task in a single batch instead of creating a job per user
func CollectJobs(ctx context.Context, db *sql.DB, now time.Time) ([]Job, error) {
	ei, err := FindEditorialInterval(ctx, db, now)
	pubDate := ei.Next
	if err != nil {
		return nil, err
	}
	if pubDate.Sub(now) > 10*time.Minute {
		fmt.Printf("Not yet close enough to the next schedule %s. Skipping.\n", pubDate)
		return []Job{}, nil
	}
	fmt.Printf("Prepare for next schedule: %s\n", pubDate)

	// Collect user IDs
	rows, err := db.QueryContext(ctx, `SELECT id FROM users;`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	var userIDs []user.UserID
	for rows.Next() {
		var uid user.UserID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var jobs []Job
	for _, uid := range userIDs {
		var newspaperID int
		err = db.QueryRowContext(ctx, `
			INSERT INTO newspapers (user_id, draft, published_at)
			VALUES ($1, TRUE, $2)
			ON CONFLICT (user_id, published_at) DO NOTHING
			RETURNING id;
		`, uid, pubDate).Scan(&newspaperID)
		if errors.Is(err, sql.ErrNoRows) {
			// TODO(#141): Handle the case where a record exists but the newspaper was not assembled due to a server outage.
			fmt.Printf(
				"The newspaper for user %d at %s already exists or is being assembled. Skipping.\n",
				uid, pubDate,
			)
			continue
		} else if err != nil {
			return nil, err
		}
		jobs = append(jobs, Job{db, uid, newspaperID})
	}

	return jobs, nil
}

type Job struct {
	DB          *sql.DB
	UserID      user.UserID
	NewspaperID int
}

func (j *Job) Timeout() time.Duration {
	return time.Minute
}

func (j *Job) Do(ctx context.Context) error {
	fmt.Printf("Assembling newspaper (ID=%d, UserID=%d)\n", j.NewspaperID, j.UserID)
	n, err := j.assembleAndPublish(ctx)
	if err != nil {
		fmt.Println("Something went wrong while assembling. Deleting.")
	} else if n == 0 {
		fmt.Println("No stories found to publish. Skipping.")
	} else {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, dErr := j.DB.ExecContext(ctx, `DELETE FROM newspapers WHERE id = $1;`, j.NewspaperID)
	if dErr != nil {
		return fmt.Errorf("%w; cleanup failed: %w", err, dErr)
	}
	return err
}

func (j *Job) assembleAndPublish(ctx context.Context) (int64, error) {
	tx, err := j.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Println(err)
		}
	}()

	res, err := tx.ExecContext(ctx, `
		UPDATE stories
		SET newspaper_id = $1
		WHERE newspaper_id IS NULL AND user_id = $2;
	`, j.NewspaperID, j.UserID)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil || n <= 0 {
		return n, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE newspapers
		SET draft = FALSE
		WHERE id = $1;
	`, j.NewspaperID)
	if err != nil {
		return n, err
	}
	err = tx.Commit()
	return n, err
}
