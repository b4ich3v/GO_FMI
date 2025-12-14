package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type ImageRecord struct {
	ID        uint64
	URL       string
	PageURL   string
	Filename  sql.NullString
	Alt       sql.NullString
	Title     sql.NullString
	Width     sql.NullInt64
	Height    sql.NullInt64
	Format    sql.NullString
	ThumbPath sql.NullString
	ThumbMIME sql.NullString
	// ThumbBlob intentionally omitted from list endpoints (can be large).
	CreatedAt time.Time
}

type ImageInsert struct {
	URL       string
	PageURL   string
	Filename  string
	Alt       string
	Title     string
	Width     int
	Height    int
	Format    string
	ThumbPath string
	ThumbMIME string
	ThumbBlob []byte
}

type SearchParams struct {
	URLContains      string
	PageURLContains  string
	FilenameContains string
	AltContains      string
	TitleContains    string
	FormatEquals     string
	MinWidth         *int
	MaxWidth         *int
	MinHeight        *int
	MaxHeight        *int
	Page             int
	PageSize         int
}

type Repository struct {
	db *sql.DB
}

func OpenMySQL(dsn string) (*Repository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// InsertImage inserts a record. Duplicate (url,page_url) is ignored.
func (r *Repository) InsertImage(ctx context.Context, in ImageInsert) error {
	if r == nil || r.db == nil {
		return errors.New("nil repository")
	}
	q := `
INSERT INTO images
  (url, page_url, filename, alt, title, width, height, format, thumb_path, thumb_mime, thumb_blob)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  -- keep first-seen metadata, but ensure thumbnail stored if missing
  thumb_blob = COALESCE(images.thumb_blob, VALUES(thumb_blob)),
  thumb_path = COALESCE(images.thumb_path, VALUES(thumb_path)),
  thumb_mime = COALESCE(images.thumb_mime, VALUES(thumb_mime)),
  width = COALESCE(images.width, VALUES(width)),
  height = COALESCE(images.height, VALUES(height)),
  format = COALESCE(images.format, VALUES(format)),
  filename = COALESCE(images.filename, VALUES(filename)),
  alt = COALESCE(images.alt, VALUES(alt)),
  title = COALESCE(images.title, VALUES(title))
`
	_, err := r.db.ExecContext(ctx, q,
		in.URL, in.PageURL,
		nullIfEmpty(in.Filename),
		nullIfEmpty(in.Alt),
		nullIfEmpty(in.Title),
		nullIntIfZero(in.Width),
		nullIntIfZero(in.Height),
		nullIfEmpty(in.Format),
		nullIfEmpty(in.ThumbPath),
		nullIfEmpty(in.ThumbMIME),
		in.ThumbBlob,
	)
	return err
}

func (r *Repository) GetImage(ctx context.Context, id uint64) (ImageRecord, error) {
	var rec ImageRecord
	q := `
SELECT id, url, page_url, filename, alt, title, width, height, format, thumb_path, thumb_mime, created_at
FROM images WHERE id = ? LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, id)
	if err := row.Scan(
		&rec.ID, &rec.URL, &rec.PageURL, &rec.Filename, &rec.Alt, &rec.Title,
		&rec.Width, &rec.Height, &rec.Format, &rec.ThumbPath, &rec.ThumbMIME, &rec.CreatedAt,
	); err != nil {
		return ImageRecord{}, err
	}
	return rec, nil
}

func (r *Repository) GetThumb(ctx context.Context, id uint64) (mime string, blob []byte, err error) {
	q := `SELECT thumb_mime, thumb_blob FROM images WHERE id = ? LIMIT 1`
	var m sql.NullString
	row := r.db.QueryRowContext(ctx, q, id)
	if err := row.Scan(&m, &blob); err != nil {
		return "", nil, err
	}
	if m.Valid {
		return m.String, blob, nil
	}
	return "application/octet-stream", blob, nil
}

func (r *Repository) Search(ctx context.Context, p SearchParams) (results []ImageRecord, total int, err error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize <= 0 || p.PageSize > 200 {
		p.PageSize = 40
	}

	where := "WHERE 1=1"
	args := []any{}

	addLike := func(field, v string) {
		if v == "" {
			return
		}
		where += fmt.Sprintf(" AND %s LIKE ?", field)
		args = append(args, "%"+v+"%")
	}
	addEq := func(field, v string) {
		if v == "" {
			return
		}
		where += fmt.Sprintf(" AND %s = ?", field)
		args = append(args, v)
	}
	addCmp := func(field, op string, v *int) {
		if v == nil {
			return
		}
		where += fmt.Sprintf(" AND %s %s ?", field, op)
		args = append(args, *v)
	}

	addLike("url", p.URLContains)
	addLike("page_url", p.PageURLContains)
	addLike("filename", p.FilenameContains)
	addLike("alt", p.AltContains)
	addLike("title", p.TitleContains)
	addEq("format", p.FormatEquals)
	addCmp("width", ">=", p.MinWidth)
	addCmp("width", "<=", p.MaxWidth)
	addCmp("height", ">=", p.MinHeight)
	addCmp("height", "<=", p.MaxHeight)

	// total
	qCount := "SELECT COUNT(*) FROM images " + where
	if err := r.db.QueryRowContext(ctx, qCount, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (p.Page - 1) * p.PageSize
	q := `
SELECT id, url, page_url, filename, alt, title, width, height, format, thumb_path, thumb_mime, created_at
FROM images
` + where + `
ORDER BY created_at DESC
LIMIT ? OFFSET ?`
	args2 := append(append([]any{}, args...), p.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, q, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var rec ImageRecord
		if err := rows.Scan(
			&rec.ID, &rec.URL, &rec.PageURL, &rec.Filename, &rec.Alt, &rec.Title,
			&rec.Width, &rec.Height, &rec.Format, &rec.ThumbPath, &rec.ThumbMIME, &rec.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, rec)
	}
	return results, total, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullIntIfZero(n int) any {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(n), Valid: true}
}
