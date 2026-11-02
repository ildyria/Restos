package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Restaurant struct {
	ID             int64    `json:"id"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Cuisine        string   `json:"cuisine"`
	Address        string   `json:"address"`
	City           string   `json:"city"`
	Website        string   `json:"website"`
	PriceLevel     *int     `json:"price_level"`
	MichelinStars  *int     `json:"michelin_stars"`
	RatingFood     *float64 `json:"rating_food"`
	RatingService  *float64 `json:"rating_service"`
	RatingAmbiance *float64 `json:"rating_ambiance"`
	RatingValue    *float64 `json:"rating_value"`
	RatingQuiet    *float64 `json:"rating_quiet"`
	Overall        *float64 `json:"overall"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	Review         string   `json:"review"`
	RankPosition   *int     `json:"rank_position"`
	CoverPhotoID   *int64   `json:"cover_photo_id"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	// Scores holds each person's ratings; the rating_* fields above are their averages.
	Scores []Score `json:"scores"`
}

// Score is one person's rating of a restaurant.
type Score struct {
	UserID         int64    `json:"user_id"`
	Name           string   `json:"name"`
	RatingFood     *float64 `json:"rating_food"`
	RatingService  *float64 `json:"rating_service"`
	RatingAmbiance *float64 `json:"rating_ambiance"`
	RatingValue    *float64 `json:"rating_value"`
	RatingQuiet    *float64 `json:"rating_quiet"`
	Overall        *float64 `json:"overall"`
	WouldReturn    *bool    `json:"would_return"`
}

// ScoreInput is the editable part of a Score.
type ScoreInput struct {
	UserID         int64    `json:"user_id"`
	RatingFood     *float64 `json:"rating_food"`
	RatingService  *float64 `json:"rating_service"`
	RatingAmbiance *float64 `json:"rating_ambiance"`
	RatingValue    *float64 `json:"rating_value"`
	RatingQuiet    *float64 `json:"rating_quiet"`
	WouldReturn    *bool    `json:"would_return"`
}

func (in ScoreInput) empty() bool {
	return in.RatingFood == nil && in.RatingService == nil && in.RatingAmbiance == nil && in.RatingValue == nil && in.RatingQuiet == nil && in.WouldReturn == nil
}

// Person is a user who has scored at least one restaurant (public: id and name only).
type Person struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RestaurantInput is the editable subset of a restaurant.
type RestaurantInput struct {
	Name           string   `json:"name"`
	Cuisine        string   `json:"cuisine"`
	Address        string   `json:"address"`
	City           string   `json:"city"`
	Website        string   `json:"website"`
	PriceLevel     *int     `json:"price_level"`
	MichelinStars  *int         `json:"michelin_stars"`
	Latitude       *float64     `json:"latitude"`
	Longitude      *float64     `json:"longitude"`
	Review         string       `json:"review"`
	Scores         []ScoreInput `json:"scores"`
	// FirstVisit is only used on creation: the visit to start the restaurant with, if any.
	FirstVisit *VisitInput `json:"first_visit,omitempty"`
	// SiteImageKey sets the website picture from a fetched (pending) image: nil leaves it
	// unchanged, "" removes it.
	SiteImageKey *string `json:"site_image_key,omitempty"`
}

type RestaurantSummary struct {
	Restaurant
	CoverKey   *string `json:"cover_key"`
	LastVisit  *string `json:"last_visit"`
	VisitCount int     `json:"visit_count"`
}

type RestaurantDetail struct {
	Restaurant
	CoverKey *string `json:"cover_key"`
	// RestaurantPhoto is the restaurant's own uploaded photo (not of a dish), if any.
	RestaurantPhoto *Photo `json:"restaurant_photo"`
	// SitePhoto is the picture taken from the restaurant's website, if any.
	SitePhoto *Photo  `json:"site_photo"`
	Visits    []Visit `json:"visits"`
}

type ListFilter struct {
	Sort    string // rank | rating | recent | name
	Query   string
	Cuisine string
}

// Category ratings are averaged over everyone who scored the restaurant (AVG skips NULLs).
const restaurantCols = `r.id, r.slug, r.name, r.cuisine, r.address, r.city, r.website, r.price_level, r.michelin_stars,
	(SELECT AVG(rating_food) FROM scores WHERE restaurant_id = r.id) AS avg_food,
	(SELECT AVG(rating_service) FROM scores WHERE restaurant_id = r.id) AS avg_service,
	(SELECT AVG(rating_ambiance) FROM scores WHERE restaurant_id = r.id) AS avg_ambiance,
	(SELECT AVG(rating_value) FROM scores WHERE restaurant_id = r.id) AS avg_value,
	(SELECT AVG(rating_quiet) FROM scores WHERE restaurant_id = r.id) AS avg_quiet,
	r.latitude, r.longitude, r.review, r.rank_position, r.cover_photo_id, r.created_at, r.updated_at`

// coverKeySQL resolves the cover: the chosen photo, else the restaurant's own photo, else the
// website picture, else the first photo of the latest visit.
const coverKeySQL = `COALESCE(
	(SELECT file_key FROM photos WHERE id = r.cover_photo_id),
	(SELECT file_key FROM photos WHERE restaurant_id = r.id AND source = 'restaurant'),
	(SELECT file_key FROM photos WHERE restaurant_id = r.id AND source = 'website'),
	(SELECT p.file_key FROM photos p JOIN visits v ON v.id = p.visit_id
	 WHERE v.restaurant_id = r.id ORDER BY v.visited_on DESC, p.position, p.id LIMIT 1))`

type scanner interface{ Scan(dest ...any) error }

func scanRestaurant(sc scanner, extra ...any) (Restaurant, error) {
	var r Restaurant
	dest := []any{&r.ID, &r.Slug, &r.Name, &r.Cuisine, &r.Address, &r.City, &r.Website, &r.PriceLevel, &r.MichelinStars,
		&r.RatingFood, &r.RatingService, &r.RatingAmbiance, &r.RatingValue, &r.RatingQuiet,
		&r.Latitude, &r.Longitude, &r.Review, &r.RankPosition, &r.CoverPhotoID, &r.CreatedAt, &r.UpdatedAt}
	if err := sc.Scan(append(dest, extra...)...); err != nil {
		return r, err
	}
	r.Overall = overall(r.RatingFood, r.RatingService, r.RatingAmbiance, r.RatingValue, r.RatingQuiet)
	r.Scores = []Score{}
	return r, nil
}

// photoBySource returns a restaurant's photo that is attached to no visit ("restaurant" or
// "website"; at most one of each), or nil.
func (s *Store) photoBySource(restaurantID int64, source string) (*Photo, error) {
	p, err := scanPhoto(s.DB.QueryRow(`SELECT `+photoCols+` FROM photos p WHERE p.restaurant_id = ? AND p.source = ?`, restaurantID, source))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func overall(vals ...*float64) *float64 {
	var sum float64
	var n int
	for _, v := range vals {
		if v != nil {
			sum += *v
			n++
		}
	}
	if n == 0 {
		return nil
	}
	avg := sum / float64(n)
	return &avg
}

func (s *Store) ListRestaurants(f ListFilter) ([]RestaurantSummary, error) {
	q := `SELECT ` + restaurantCols + `, ` + coverKeySQL + `,
		(SELECT MAX(visited_on) FROM visits WHERE restaurant_id = r.id),
		(SELECT COUNT(*) FROM visits WHERE restaurant_id = r.id)
		FROM restaurants r WHERE 1=1`
	var args []any
	if f.Query != "" {
		q += ` AND (r.name LIKE ? OR r.cuisine LIKE ? OR r.city LIKE ?)`
		like := "%" + f.Query + "%"
		args = append(args, like, like, like)
	}
	if f.Cuisine != "" {
		q += ` AND r.cuisine = ?`
		args = append(args, f.Cuisine)
	}
	const overallSQL = `((COALESCE(avg_food,0)+COALESCE(avg_service,0)+COALESCE(avg_ambiance,0)+COALESCE(avg_value,0)+COALESCE(avg_quiet,0)) /
		NULLIF((avg_food IS NOT NULL)+(avg_service IS NOT NULL)+(avg_ambiance IS NOT NULL)+(avg_value IS NOT NULL)+(avg_quiet IS NOT NULL),0))`
	switch f.Sort {
	case "rating":
		q += ` ORDER BY ` + overallSQL + ` IS NULL, ` + overallSQL + ` DESC, r.name`
	case "recent":
		q += ` ORDER BY (SELECT MAX(visited_on) FROM visits WHERE restaurant_id = r.id) IS NULL,
			(SELECT MAX(visited_on) FROM visits WHERE restaurant_id = r.id) DESC, r.name`
	case "name":
		q += ` ORDER BY r.name COLLATE NOCASE`
	default: // rank: ranked first by position, then unranked by rating
		q += ` ORDER BY r.rank_position IS NULL, r.rank_position, ` + overallSQL + ` IS NULL, ` + overallSQL + ` DESC, r.name`
	}
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RestaurantSummary{}
	for rows.Next() {
		var sum RestaurantSummary
		r, err := scanRestaurant(rows, &sum.CoverKey, &sum.LastVisit, &sum.VisitCount)
		if err != nil {
			return nil, err
		}
		sum.Restaurant = r
		out = append(out, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	scores, err := s.scoresWhere(`1=1`)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if sc, ok := scores[out[i].ID]; ok {
			out[i].Scores = sc
		}
	}
	return out, nil
}

// scoresWhere loads per-person scores matching a condition on s (scores) and u (users), keyed by restaurant.
func (s *Store) scoresWhere(where string, args ...any) (map[int64][]Score, error) {
	rows, err := s.DB.Query(`SELECT s.restaurant_id, s.user_id, u.name, s.rating_food, s.rating_service, s.rating_ambiance,
		s.rating_value, s.rating_quiet, s.would_return FROM scores s JOIN users u ON u.id = s.user_id WHERE `+where+` ORDER BY u.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]Score{}
	for rows.Next() {
		var rid int64
		var sc Score
		if err := rows.Scan(&rid, &sc.UserID, &sc.Name, &sc.RatingFood, &sc.RatingService, &sc.RatingAmbiance,
			&sc.RatingValue, &sc.RatingQuiet, &sc.WouldReturn); err != nil {
			return nil, err
		}
		sc.Overall = overall(sc.RatingFood, sc.RatingService, sc.RatingAmbiance, sc.RatingValue, sc.RatingQuiet)
		out[rid] = append(out[rid], sc)
	}
	return out, rows.Err()
}

// People lists the users who have scored at least one restaurant.
func (s *Store) People() ([]Person, error) {
	rows, err := s.DB.Query(`SELECT id, name FROM users WHERE id IN (SELECT user_id FROM scores) ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Person{}
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CountRestaurants() (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM restaurants`).Scan(&n)
	return n, err
}

func (s *Store) Cuisines() ([]string, error) {
	rows, err := s.DB.Query(`SELECT DISTINCT cuisine FROM restaurants WHERE cuisine <> '' ORDER BY cuisine COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) RestaurantBySlug(slug string) (RestaurantDetail, error) {
	return s.restaurantDetail(`r.slug = ?`, slug)
}

func (s *Store) RestaurantByID(id int64) (RestaurantDetail, error) {
	return s.restaurantDetail(`r.id = ?`, id)
}

func (s *Store) restaurantDetail(where string, arg any) (RestaurantDetail, error) {
	var d RestaurantDetail
	row := s.DB.QueryRow(`SELECT `+restaurantCols+`, `+coverKeySQL+` FROM restaurants r WHERE `+where, arg)
	r, err := scanRestaurant(row, &d.CoverKey)
	if err != nil {
		return d, notFound(err)
	}
	d.Restaurant = r
	scores, err := s.scoresWhere(`s.restaurant_id = ?`, r.ID)
	if err != nil {
		return d, err
	}
	if sc, ok := scores[r.ID]; ok {
		d.Scores = sc
	}
	if d.RestaurantPhoto, err = s.photoBySource(r.ID, "restaurant"); err != nil {
		return d, err
	}
	if d.SitePhoto, err = s.photoBySource(r.ID, "website"); err != nil {
		return d, err
	}
	d.Visits, err = s.visitsFor(r.ID)
	return d, err
}

func (s *Store) CreateRestaurant(in RestaurantInput) (int64, error) {
	slug, err := s.uniqueSlug(in.Name, 0)
	if err != nil {
		return 0, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO restaurants(slug, name, cuisine, address, city, website, price_level, michelin_stars,
		latitude, longitude, review) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		slug, in.Name, in.Cuisine, in.Address, in.City, in.Website, in.PriceLevel, in.MichelinStars,
		in.Latitude, in.Longitude, in.Review)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := replaceScores(tx, id, in.Scores); err != nil {
		return 0, err
	}
	if in.FirstVisit != nil {
		if _, err := createVisit(tx, id, *in.FirstVisit); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

func (s *Store) UpdateRestaurant(id int64, in RestaurantInput) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// The slug is kept stable after creation so shared links don't break.
	res, err := tx.Exec(`UPDATE restaurants SET name=?, cuisine=?, address=?, city=?, website=?, price_level=?, michelin_stars=?,
		latitude=?, longitude=?, review=?, updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		in.Name, in.Cuisine, in.Address, in.City, in.Website, in.PriceLevel, in.MichelinStars,
		in.Latitude, in.Longitude, in.Review, id)
	if err != nil {
		return err
	}
	if err := affected(res); err != nil {
		return err
	}
	if err := replaceScores(tx, id, in.Scores); err != nil {
		return err
	}
	return tx.Commit()
}

// ErrUnknownUser is returned when a score refers to a user that does not exist.
var ErrUnknownUser = errors.New("unknown user")

// replaceScores swaps a restaurant's scores for the given set; entries with nothing filled in are dropped.
func replaceScores(tx *sql.Tx, restaurantID int64, scores []ScoreInput) error {
	if _, err := tx.Exec(`DELETE FROM scores WHERE restaurant_id = ?`, restaurantID); err != nil {
		return err
	}
	for _, sc := range scores {
		if sc.empty() {
			continue
		}
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, sc.UserID).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("%w %d", ErrUnknownUser, sc.UserID)
		}
		if _, err := tx.Exec(`INSERT INTO scores(restaurant_id, user_id, rating_food, rating_service, rating_ambiance, rating_value, rating_quiet, would_return)
			VALUES(?,?,?,?,?,?,?,?)`, restaurantID, sc.UserID, sc.RatingFood, sc.RatingService, sc.RatingAmbiance, sc.RatingValue, sc.RatingQuiet, sc.WouldReturn); err != nil {
			return err
		}
	}
	return nil
}

// DeleteRestaurant removes the restaurant and returns the file keys of its photos for cleanup.
func (s *Store) DeleteRestaurant(id int64) ([]string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	keys, err := queryKeys(tx, `SELECT file_key FROM photos WHERE restaurant_id = ?`, id)
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(`DELETE FROM restaurants WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	if err := affected(res); err != nil {
		return nil, err
	}
	return keys, tx.Commit()
}

// SetRanking rewrites rank positions: listed ids get 1..n, everything else becomes unranked.
func (s *Store) SetRanking(ids []int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE restaurants SET rank_position = NULL`); err != nil {
		return err
	}
	for i, id := range ids {
		res, err := tx.Exec(`UPDATE restaurants SET rank_position = ? WHERE id = ?`, i+1, id)
		if err != nil {
			return err
		}
		if err := affected(res); err != nil {
			return fmt.Errorf("restaurant %d: %w", id, err)
		}
	}
	return tx.Commit()
}

// SetCover picks the restaurant's cover among its photos; nil goes back to automatic.
func (s *Store) SetCover(restaurantID int64, photoID *int64) error {
	if photoID != nil {
		var n int
		err := s.DB.QueryRow(`SELECT COUNT(*) FROM photos WHERE id = ? AND restaurant_id = ?`, *photoID, restaurantID).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
	}
	res, err := s.DB.Exec(`UPDATE restaurants SET cover_photo_id = ? WHERE id = ?`, photoID, restaurantID)
	if err != nil {
		return err
	}
	return affected(res)
}

func (s *Store) uniqueSlug(name string, excludeID int64) (string, error) {
	base := slugify(name)
	slug := base
	for i := 2; ; i++ {
		var id int64
		err := s.DB.QueryRow(`SELECT id FROM restaurants WHERE slug = ?`, slug).Scan(&id)
		if err == sql.ErrNoRows || id == excludeID {
			return slug, nil
		}
		if err != nil {
			return "", err
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case strings.ContainsRune("àáâãäå", r):
			b.WriteByte('a')
			dash = false
		case strings.ContainsRune("èéêë", r):
			b.WriteByte('e')
			dash = false
		case strings.ContainsRune("ìíîï", r):
			b.WriteByte('i')
			dash = false
		case strings.ContainsRune("òóôõö", r):
			b.WriteByte('o')
			dash = false
		case strings.ContainsRune("ùúûü", r):
			b.WriteByte('u')
			dash = false
		case r == 'ç':
			b.WriteByte('c')
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "restaurant"
	}
	return out
}

type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func queryKeys(q queryer, query string, args ...any) ([]string, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
