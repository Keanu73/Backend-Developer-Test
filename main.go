package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var db *pgx.Conn
var err error

// Database settings
const (
	host     = "localhost"
	port     = 5432 // Default port
	user     = "postgres"
	password = "password"
	dbname   = "spots"
)

// Spot struct - defines the properties of a spot.
type Spot struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Website     *string     `json:"website"`
	Coordinates pgtype.Text `json:"coordinates"`
	// Latitude    float32       `json:"latitude"`
	// Longitude   float32       `json:"longitude"`
	// Distance    float32       `json:"distance,omitempty"`
	Description *string       `json:"description"`
	Rating      pgtype.Float8 `json:"rating"`
	DomainCount int           `json:"domain_count,omitempty"`
}

// Spots struct - slice used for bundling multiple spots.
type Spots struct {
	Spots []Spot `json:"spots"`
	Total int    `json:"total"`
}

// SpotAreaQuery struct - used to store parameters for query of spots within area.
type SpotAreaQuery struct {
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
	Radius    float32 `json:"radius"`
	// Type      string  `json:"type"`
}

// Connect function
func Connect(ctx context.Context) error {
	db, err = pgx.Connect(
		ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, port, dbname),
	)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	app := Bootstrap()

	log.Fatal(app.Listen(":3000"))
}

func Bootstrap() *fiber.App {
	ctx := context.Background()

	// Connect with database
	if err := Connect(ctx); err != nil {
		log.Fatal(err)
	}

	// Create a Fiber app
	app := fiber.New()

	// Get duplicate spots
	app.Get(
		"/spots/duplicates", func(c *fiber.Ctx) error {
			// Select amount of spots which have more than 1 duplicate domain

			// Let me walk you through this SQL query:

			// SELECT DISTINCT ON (website) *, COUNT(*) OVER (PARTITION BY website) AS domain_count FROM spots s
			// * This line is wrapped in something called a CTE,
			// * or a sub-query. Why? Because we want all the spot's row data,
			// * as well as the amount of spots with the same, duplicate domain for each row.

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

				if err := rows.Scan(
					&spot.ID, &spot.Name, &spot.Website, &spot.Coordinates, &spot.Description,
					&spot.Rating, &spot.DomainCount,
				); err != nil {
					return fmt.Errorf("spot: error scanning: %w", err)
				}

				// Append scanned spot into slice
				result.Spots = append(result.Spots, spot)
				result.Total = len(result.Spots)
			}
			// Return spots in JSON format
			return c.JSON(result)
		},
	)

	app.Get(
		"/spots/inArea", func(c *fiber.Ctx) error {
			// Selects spots within a certain geographic area.
			// 3 parameters:
			// * Latitude - float
			// * Longitude - float
			// * Radius (in meters - float)
			// deprecated:
			// * Type - string: "circle" or "square" (not necessary)

			query := new(SpotAreaQuery)

			// Parse body into struct
			// lazy validation technique. it works though!
			if err := c.BodyParser(query); err != nil {
				return c.Status(400).SendString(fmt.Sprintf("unable to parse body: %s", err.Error()))
			}

			// validates only circle/square types
			// if query.Type != "circle" && query.Type != "square" {
			//	return c.Status(400).SendString("invalid type")
			// }

			// Query DB

			/* if we wanted to chuck in latitude/longitude/distance:
			   ST_Y(coordinates::geometry) AS latitude,
			   ST_X(coordinates::geometry) AS longitude,
			   ST_Distance_Sphere(coordinates, ST_MakePoint($1, $2)) AS distance,
			*/
			dbQuery := `SELECT id, name, website, ST_AsText(coordinates), description, rating FROM spots
    WHERE ST_DWithin(ST_MakePoint($1, $2)::geography, coordinates, $3)
		ORDER BY ST_Distance(ST_MakePoint($1, $2)::geography, coordinates), rating;
    `

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

			/*
				if query.Type == "circle" {
					dbQuery += `
					WHERE ST_DWithin(ST_MakePoint($1, $2)::geometry, coordinates::geometry, $3)
				  ORDER BY ST_Distance_Sphere(coordinates, ST_MakePoint($1, $2)::geography);`
				} else {
					dbQuery += `
					WHERE ST_DWithin(ST_MakePoint($1, $2)::geography, coordinates, $3)
				  ORDER BY ST_Distance_Sphere(coordinates, ST_MakePoint($1, $2)::geography);`
				}
			*/

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

				if err := rows.Scan(
					&spot.ID, &spot.Name, &spot.Website, &spot.Coordinates, &spot.Description, &spot.Rating,
				); err != nil {
					return fmt.Errorf("spot: error scanning: %w", err)
				}

				// Append scanned spot into slice
				result.Spots = append(result.Spots, spot)
				result.Total = len(result.Spots)
			}

			return c.JSON(result)
		},
	)

	return app
}
