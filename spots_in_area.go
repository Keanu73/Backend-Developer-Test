package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

/*
			* Based on my Stackoverflow reading, I've concluded that there's no point
			* trying to do different WHERE queries based on either a square or circle area.
			* I'm a sucker for optimisation, so I decided to investigate
			* the fastest & easiest way to find spots within a radius of a point.

					* firstly, the PostGIS docs say:
				http://postgis.net/docs/PostGIS_FAQ.html
				 9.14.
				What is the best way to find all objects within a radius of another object?

				To use the database most efficiently, it is best to do radius queries which combine the radius test with a bounding box test: the bounding box test uses the spatial index, giving fast access to a subset of data which the radius test is then applied to.

				The ST_DWithin(geometry, geometry, distance) function is a handy way of performing an indexed distance search. It works by creating a search rectangle large enough to enclose the distance radius, then performing an exact distance search on the indexed subset of results.

				For example, to find all objects with 100 meters of POINT(1000 1000) the following query would work well:

				SELECT * FROM geotable
				WHERE ST_DWithin(geocolumn, 'POINT(1000 1000)', 100.0);

			* We also have a few options other than using DWithin:
					* 1) Use ST_DistanceSphere(ST_MakePoint(-2.91027, 53.3857)::geometry, coordinates::geometry) <= 30000
					* 2) Use ST_Intersects(coordinates, ST_Buffer(ST_Point(-2.91027, 53.3857)::geography, 1000))
					* 3) Use ST_Expands along with coordinates to test whether coordinates are within boundary box.

			However, none of these options were any faster. ST_Expands was in fact a function previously used by DWithin,
			but DWithin uses a "faster short-circuit distance function".
			This GIS StackExchange user found just that: https://gis.stackexchange.com/questions/247113/setting-up-indexes-for-postgis-distance-queries
			(you also need an index on the geography column for DWithin to truly use its' power).

			Back to the original question: is it worth it to use ST_Expands/ST_Buffer to create a square/circle just for
			the sake of checking whether the points lay in that circle?
			The answer: probably not. As one GIS Stackexchange user said:
			"Since your search region is circular, it is perhaps best not to consider it to be a polygon, but as a point with a radius and to use the ST_DWithin() function:"
			"It should save you and the processor a lot of effort."
			https://gis.stackexchange.com/a/87679

	  It is possible to also use an Envelope search (ST_Envelope creates a bounding box),
		but comparing distances between two points is much cheaper than creating a whole new bounding box and
		performing more calculations to check whether the spots we're looking for lay in this box - when...
		we can just calculate it based on distance from the original point.
*/

func SpotsInArea(ctx context.Context) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Selects spots within a certain geographic area.
		// 3 parameters:
		// * Latitude - float
		// * Longitude - float
		// * Radius (in meters - float)
		// deprecated:
		// * Type - string: "circle" or "square" (not necessary)

		query := new(SpotAreaQuery)

		// Parse body into struct
		if err := c.BodyParser(query); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				fiber.Map{
					"message": err.Error(),
				},
			)
		}

		errors := ValidateStruct(*query)
		if errors != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errors)
		}

		// Query DB
		dbQuery := `
			SELECT id, name, website, ST_AsText(coordinates), description, rating, 
			ST_Distance(ST_MakePoint($1,
			$2)::geography, coordinates) AS distance FROM spots
    WHERE ST_DWithin(ST_MakePoint($1, $2)::geography, coordinates, $3)
		ORDER BY distance;`

		rows, err := db.Query(
			ctx,
			dbQuery, query.Longitude, query.Latitude, query.Radius,
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
				&spot.ID, &spot.Name, &spot.Website, &coordinateText, &spot.Description, &ratingFloat, &spot.Distance,
			); err != nil {
				return fmt.Errorf("spot: error scanning: %w", err)
			}

			spot.Coordinates = coordinateText.String
			spot.Rating = ratingFloat.Float64

			// Append scanned spot into slice
			result.Spots = append(result.Spots, spot)
			result.Total = len(result.Spots)
		}

		// Uses SliceStable to preserve original distance order
		sort.SliceStable(
			result.Spots,
			func(i, j int) bool {
				// We'll have to shove the logic into a variable.
				// Basically, checks that:
				// 2nd spots' distance - 1st spots distance < 50m
				// That the 2nd spot's rating is greater than the first spot's rating.
				b := ((result.Spots[i].Distance - result.Spots[j].Distance) < 50) && (result.Spots[i].Rating > result.
					Spots[j].Rating)
				return b
			},
		)

		return c.JSON(result)
	}
}
