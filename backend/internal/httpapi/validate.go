package httpapi

import (
	"math"
	"net/mail"
	"strings"
	"time"

	"restos/internal/db"
)

func checkRating(name string, v *float64) error {
	if v == nil {
		return nil
	}
	if *v < 0.5 || *v > 5 || math.Mod(*v*2, 1) != 0 {
		return badRequest(name + " must be between 0.5 and 5 in steps of 0.5")
	}
	return nil
}

func cleanRestaurant(in *db.RestaurantInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Cuisine = strings.TrimSpace(in.Cuisine)
	in.Address = strings.TrimSpace(in.Address)
	in.City = strings.TrimSpace(in.City)
	in.Website = strings.TrimSpace(in.Website)
	in.Review = strings.TrimSpace(in.Review)
	if in.Name == "" || len(in.Name) > 200 {
		return badRequest("name is required (max 200 characters)")
	}
	if len(in.Review) > 20000 {
		return badRequest("review is too long")
	}
	if in.Website != "" && !strings.HasPrefix(in.Website, "http://") && !strings.HasPrefix(in.Website, "https://") {
		return badRequest("website must start with http:// or https://")
	}
	if in.PriceLevel != nil && (*in.PriceLevel < 1 || *in.PriceLevel > 4) {
		return badRequest("price_level must be between 1 and 4")
	}
	if in.MichelinStars != nil && (*in.MichelinStars < 1 || *in.MichelinStars > 3) {
		return badRequest("michelin_stars must be between 1 and 3")
	}
	if (in.Latitude == nil) != (in.Longitude == nil) {
		return badRequest("latitude and longitude must be set together")
	}
	if in.SiteImageKey != nil && *in.SiteImageKey != "" && !mediaKey.MatchString(*in.SiteImageKey) {
		return badRequest("invalid site_image_key")
	}
	if in.Latitude != nil && (*in.Latitude < -90 || *in.Latitude > 90 || *in.Longitude < -180 || *in.Longitude > 180) {
		return badRequest("latitude must be within ±90 and longitude within ±180")
	}
	seen := map[int64]bool{}
	for _, sc := range in.Scores {
		if seen[sc.UserID] {
			return badRequest("duplicate person in scores")
		}
		seen[sc.UserID] = true
		for name, v := range map[string]*float64{
			"rating_food": sc.RatingFood, "rating_service": sc.RatingService,
			"rating_ambiance": sc.RatingAmbiance, "rating_value": sc.RatingValue, "rating_quiet": sc.RatingQuiet,
		} {
			if err := checkRating(name, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanVisit(in *db.VisitInput) error {
	in.Notes = strings.TrimSpace(in.Notes)
	if _, err := time.Parse("2006-01-02", in.VisitedOn); err != nil {
		return badRequest("visited_on must be a date (YYYY-MM-DD)")
	}
	if in.Meal != "lunch" && in.Meal != "dinner" {
		return badRequest("meal must be lunch or dinner")
	}
	return nil
}

var courses = map[string]bool{"starter": true, "main": true, "dessert": true, "drink": true, "other": true}

func cleanDish(in *db.DishInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.Name == "" || len(in.Name) > 200 {
		return badRequest("dish name is required (max 200 characters)")
	}
	if in.Course != nil && *in.Course == "" {
		in.Course = nil
	}
	if in.Course != nil && !courses[*in.Course] {
		return badRequest("invalid course")
	}
	return nil
}

func cleanEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if _, err := mail.ParseAddress(email); err != nil || strings.ContainsAny(email, " <>") {
		return "", badRequest("invalid email address")
	}
	return email, nil
}
