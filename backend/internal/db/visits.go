package db

import "database/sql"

type Photo struct {
	ID       int64  `json:"id"`
	DishID   int64  `json:"dish_id"`
	Key      string `json:"key"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Position int    `json:"position"`
}

type Dish struct {
	ID       int64   `json:"id"`
	VisitID  int64   `json:"visit_id"`
	Name     string  `json:"name"`
	Course   *string `json:"course"`
	Notes    string  `json:"notes"`
	Position int     `json:"position"`
	Photos   []Photo `json:"photos"`
}

type Visit struct {
	ID           int64  `json:"id"`
	RestaurantID int64  `json:"restaurant_id"`
	VisitedOn    string `json:"visited_on"`
	Meal         string `json:"meal"`
	Notes        string `json:"notes"`
	Dishes       []Dish `json:"dishes"`
}

type VisitInput struct {
	VisitedOn string `json:"visited_on"`
	Meal      string `json:"meal"`
	Notes     string `json:"notes"`
}

type DishInput struct {
	Name   string  `json:"name"`
	Course *string `json:"course"`
	Notes  string  `json:"notes"`
}

// visitsFor loads all visits (newest first) with dishes and photos for a restaurant.
func (s *Store) visitsFor(restaurantID int64) ([]Visit, error) {
	visits := []Visit{}
	byVisit := map[int64]int{}
	rows, err := s.DB.Query(`SELECT id, restaurant_id, visited_on, meal, notes FROM visits
		WHERE restaurant_id = ? ORDER BY visited_on DESC, id DESC`, restaurantID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		v := Visit{Dishes: []Dish{}}
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.VisitedOn, &v.Meal, &v.Notes); err != nil {
			rows.Close()
			return nil, err
		}
		byVisit[v.ID] = len(visits)
		visits = append(visits, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(visits) == 0 {
		return visits, nil
	}

	dishIdx := map[int64][2]int{} // dish id -> [visit index, dish index]
	drows, err := s.DB.Query(`SELECT d.id, d.visit_id, d.name, d.course, d.notes, d.position
		FROM dishes d JOIN visits v ON v.id = d.visit_id WHERE v.restaurant_id = ? ORDER BY d.position, d.id`, restaurantID)
	if err != nil {
		return nil, err
	}
	for drows.Next() {
		d := Dish{Photos: []Photo{}}
		if err := drows.Scan(&d.ID, &d.VisitID, &d.Name, &d.Course, &d.Notes, &d.Position); err != nil {
			drows.Close()
			return nil, err
		}
		vi := byVisit[d.VisitID]
		dishIdx[d.ID] = [2]int{vi, len(visits[vi].Dishes)}
		visits[vi].Dishes = append(visits[vi].Dishes, d)
	}
	drows.Close()
	if err := drows.Err(); err != nil {
		return nil, err
	}

	prows, err := s.DB.Query(`SELECT p.id, p.dish_id, p.file_key, p.width, p.height, p.position
		FROM photos p JOIN dishes d ON d.id = p.dish_id JOIN visits v ON v.id = d.visit_id
		WHERE v.restaurant_id = ? ORDER BY p.position, p.id`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var p Photo
		if err := prows.Scan(&p.ID, &p.DishID, &p.Key, &p.Width, &p.Height, &p.Position); err != nil {
			return nil, err
		}
		ix := dishIdx[p.DishID]
		d := &visits[ix[0]].Dishes[ix[1]]
		d.Photos = append(d.Photos, p)
	}
	return visits, prows.Err()
}

func (s *Store) CreateVisit(restaurantID int64, in VisitInput) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO visits(restaurant_id, visited_on, meal, notes) VALUES(?,?,?,?)`,
		restaurantID, in.VisitedOn, in.Meal, in.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateVisit(id int64, in VisitInput) error {
	res, err := s.DB.Exec(`UPDATE visits SET visited_on=?, meal=?, notes=? WHERE id=?`, in.VisitedOn, in.Meal, in.Notes, id)
	if err != nil {
		return err
	}
	return affected(res)
}

func (s *Store) VisitByID(id int64) (Visit, error) {
	var v Visit
	err := s.DB.QueryRow(`SELECT id, restaurant_id, visited_on, meal, notes FROM visits WHERE id=?`, id).
		Scan(&v.ID, &v.RestaurantID, &v.VisitedOn, &v.Meal, &v.Notes)
	return v, notFound(err)
}

func (s *Store) DeleteVisit(id int64) ([]string, error) {
	return s.deleteWithKeys(`DELETE FROM visits WHERE id=?`,
		`SELECT p.file_key FROM photos p JOIN dishes d ON d.id = p.dish_id WHERE d.visit_id = ?`, id)
}

func (s *Store) CreateDish(visitID int64, in DishInput) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO dishes(visit_id, name, course, notes, position)
		VALUES(?,?,?,?, (SELECT COALESCE(MAX(position),0)+1 FROM dishes WHERE visit_id = ?))`,
		visitID, in.Name, in.Course, in.Notes, visitID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateDish(id int64, in DishInput) error {
	res, err := s.DB.Exec(`UPDATE dishes SET name=?, course=?, notes=? WHERE id=?`, in.Name, in.Course, in.Notes, id)
	if err != nil {
		return err
	}
	return affected(res)
}

func (s *Store) DishByID(id int64) (Dish, error) {
	var d Dish
	err := s.DB.QueryRow(`SELECT id, visit_id, name, course, notes, position FROM dishes WHERE id=?`, id).
		Scan(&d.ID, &d.VisitID, &d.Name, &d.Course, &d.Notes, &d.Position)
	return d, notFound(err)
}

func (s *Store) DeleteDish(id int64) ([]string, error) {
	return s.deleteWithKeys(`DELETE FROM dishes WHERE id=?`, `SELECT file_key FROM photos WHERE dish_id = ?`, id)
}

func (s *Store) AddPhoto(dishID int64, key string, w, h int) (Photo, error) {
	res, err := s.DB.Exec(`INSERT INTO photos(dish_id, file_key, width, height, position)
		VALUES(?,?,?,?, (SELECT COALESCE(MAX(position),0)+1 FROM photos WHERE dish_id = ?))`,
		dishID, key, w, h, dishID)
	if err != nil {
		return Photo{}, err
	}
	id, _ := res.LastInsertId()
	var p Photo
	err = s.DB.QueryRow(`SELECT id, dish_id, file_key, width, height, position FROM photos WHERE id=?`, id).
		Scan(&p.ID, &p.DishID, &p.Key, &p.Width, &p.Height, &p.Position)
	return p, err
}

func (s *Store) DeletePhoto(id int64) ([]string, error) {
	return s.deleteWithKeys(`DELETE FROM photos WHERE id=?`, `SELECT file_key FROM photos WHERE id = ?`, id)
}

// ReorderPhotos sets photo positions to the order of ids.
func (s *Store) ReorderPhotos(ids []int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		res, err := tx.Exec(`UPDATE photos SET position = ? WHERE id = ?`, i+1, id)
		if err != nil {
			return err
		}
		if err := affected(res); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// deleteWithKeys collects photo file keys, runs the delete, then clears any dangling cover
// references, all in one transaction. The caller removes the returned files from disk.
func (s *Store) deleteWithKeys(deleteSQL, keysSQL string, id int64) ([]string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	keys, err := queryKeys(tx, keysSQL, id)
	if err != nil {
		return nil, err
	}
	res, err := tx.Exec(deleteSQL, id)
	if err != nil {
		return nil, err
	}
	if err := affected(res); err != nil {
		return nil, err
	}
	if err := clearDanglingCovers(tx); err != nil {
		return nil, err
	}
	return keys, tx.Commit()
}

func clearDanglingCovers(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE restaurants SET cover_photo_id = NULL
		WHERE cover_photo_id IS NOT NULL AND cover_photo_id NOT IN (SELECT id FROM photos)`)
	return err
}

func (s *Store) PhotoByID(id int64) (Photo, error) {
	var p Photo
	err := s.DB.QueryRow(`SELECT id, dish_id, file_key, width, height, position FROM photos WHERE id=?`, id).
		Scan(&p.ID, &p.DishID, &p.Key, &p.Width, &p.Height, &p.Position)
	return p, notFound(err)
}
