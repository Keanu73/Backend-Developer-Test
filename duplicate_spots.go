package main

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

func DuplicateSpots(ctx context.Context) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Select amount of spots which have more than 1 duplicate domain

		// Let me walk you through this SQL query:

		// SELECT DISTINCT ON (website) *, COUNT(*) OVER (PARTITION BY website) AS domain_count FROM spots s
		// * This line is wrapped in something called a CTE,
		// * or a sub-query. Why? Because we want all the spot's row data,
		// * as well as the amount of spots with the same, duplicate domain for each row.
		// * We're using a window function to get the amount of rows with the same domain name.

		// SELECT id, name, website, ST_AsText(coordinates),
		//				description, rating, domain_count
		//				FROM cte WHERE (website IS NOT NULL AND website <> '' AND domain_count > 1);
		// * This line now queries the previous query (with the domain_count row), and filters that out by:
		// * re-selecting all the fields & using PostGIS to convert hex coordinate value into latitude/longitude.
		// * making sure the website field is not NULL or ''
		// * making sure that there is more than 1 row with the same domain.

		rows, err := db.Query(
			ctx,
			`
				WITH cte AS (SELECT DISTINCT ON (website) *, 
				COUNT(*) OVER (PARTITION BY website) AS domain_count FROM spots s)
				
				SELECT id, name, website, ST_AsText(coordinates), description, 
				rating, domain_count FROM cte WHERE (website IS NOT NULL AND website <> '' AND domain_count > 1);
				`,
		)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		defer rows.Close()
		result := Spots{}

		for rows.Next() {
			spot := Spot{}

			var coordinateText pgtype.Text
			var ratingFloat pgtype.Float8

			if err := rows.Scan(
				&spot.ID, &spot.Name, &spot.Website, &coordinateText, &spot.Description,
				&ratingFloat, &spot.DomainCount,
			); err != nil {
				return fmt.Errorf("spot: error scanning: %w", err)
			}

			spot.Coordinates = coordinateText.String
			spot.Rating = ratingFloat.Float64

			// Append scanned spot into slice
			result.Spots = append(result.Spots, spot)
			result.Total = len(result.Spots)
		}
		// Return spots in JSON format
		return c.JSON(result)
	}
}
