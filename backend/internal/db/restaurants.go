package db

import (
	"database/sql"
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
	Overall        *float64 `json:"overall"`
	Review         string   `json:"review"`
	RankPosition   *int     `json:"rank_position"`
	CoverPhotoID   *int64   `json:"cover_photo_id"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// RestaurantInput is the editable subset of a restaurant.
type RestaurantInput struct {
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
	Review         string   `json:"review"`
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
	Visits   []Visit `json:"visits"`
}

type ListFilter struct {
	Sort    string // rank | rating | recent | name
	Query   string
	Cuisine string
}

const restaurantCols = `r.id, r.slug, r.name, r.cuisine, r.address, r.city, r.website, r.price_level, r.michelin_stars,
	r.rating_food, r.rating_service, r.rating_ambiance, r.rating_value, r.review,
	r.rank_position, r.cover_photo_id, r.created_at, r.updated_at`

// coverKeySQL resolves the cover: the chosen photo, else the first photo of the latest visit.
const coverKeySQL = `COALESCE(
	(SELECT file_key FROM photos WHERE id = r.cover_photo_id),
	(SELECT p.file_key FROM photos p JOIN dishes d ON d.id = p.dish_id JOIN visits v ON v.id = d.visit_id
	 WHERE v.restaurant_id = r.id ORDER BY v.visited_on DESC, d.position, p.position LIMIT 1))`

type scanner interface{ Scan(dest ...any) error }

func scanRestaurant(sc scanner, extra ...any) (Restaurant, error) {
	var r Restaurant
	dest := []any{&r.ID, &r.Slug, &r.Name, &r.Cuisine, &r.Address, &r.City, &r.Website, &r.PriceLevel, &r.MichelinStars,
		&r.RatingFood, &r.RatingService, &r.RatingAmbiance, &r.RatingValue, &r.Review,
		&r.RankPosition, &r.CoverPhotoID, &r.CreatedAt, &r.UpdatedAt}
	if err := sc.Scan(append(dest, extra...)...); err != nil {
		return r, err
	}
	r.Overall = overall(r.RatingFood, r.RatingService, r.RatingAmbiance, r.RatingValue)
	return r, nil
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
	const overallSQL = `((COALESCE(r.rating_food,0)+COALESCE(r.rating_service,0)+COALESCE(r.rating_ambiance,0)+COALESCE(r.rating_value,0)) /
		NULLIF((r.rating_food IS NOT NULL)+(r.rating_service IS NOT NULL)+(r.rating_ambiance IS NOT NULL)+(r.rating_value IS NOT NULL),0))`
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
	return out, rows.Err()
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
	d.Visits, err = s.visitsFor(r.ID)
	return d, err
}

func (s *Store) CreateRestaurant(in RestaurantInput) (int64, error) {
	slug, err := s.uniqueSlug(in.Name, 0)
	if err != nil {
		return 0, err
	}
	res, err := s.DB.Exec(`INSERT INTO restaurants(slug, name, cuisine, address, city, website, price_level, michelin_stars,
		rating_food, rating_service, rating_ambiance, rating_value, review) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		slug, in.Name, in.Cuisine, in.Address, in.City, in.Website, in.PriceLevel, in.MichelinStars,
		in.RatingFood, in.RatingService, in.RatingAmbiance, in.RatingValue, in.Review)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateRestaurant(id int64, in RestaurantInput) error {
	// The slug is kept stable after creation so shared links don't break.
	res, err := s.DB.Exec(`UPDATE restaurants SET name=?, cuisine=?, address=?, city=?, website=?, price_level=?, michelin_stars=?,
		rating_food=?, rating_service=?, rating_ambiance=?, rating_value=?, review=?,
		updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		in.Name, in.Cuisine, in.Address, in.City, in.Website, in.PriceLevel, in.MichelinStars,
		in.RatingFood, in.RatingService, in.RatingAmbiance, in.RatingValue, in.Review, id)
	if err != nil {
		return err
	}
	return affected(res)
}

// DeleteRestaurant removes the restaurant and returns the file keys of its photos for cleanup.
func (s *Store) DeleteRestaurant(id int64) ([]string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	keys, err := queryKeys(tx, `SELECT p.file_key FROM photos p JOIN dishes d ON d.id = p.dish_id
		JOIN visits v ON v.id = d.visit_id WHERE v.restaurant_id = ?`, id)
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

func (s *Store) SetCover(restaurantID, photoID int64) error {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM photos p JOIN dishes d ON d.id = p.dish_id
		JOIN visits v ON v.id = d.visit_id WHERE p.id = ? AND v.restaurant_id = ?`, photoID, restaurantID).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
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
