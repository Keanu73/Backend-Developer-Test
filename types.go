package main

// Spot struct - defines the properties of a spot.
type Spot struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Website     *string `json:"website"`
	Coordinates string  `json:"coordinates"`
	// Latitude    float32       `json:"latitude"`
	// Longitude   float32       `json:"longitude"`
	Distance    float64 `json:"distance,omitempty"`
	Description *string `json:"description"`
	Rating      float64 `json:"rating"`
	DomainCount int     `json:"domain_count,omitempty"`
}

// Spots struct - slice used for bundling multiple spots.
type Spots struct {
	Spots []Spot `json:"spots"`
	Total int    `json:"total"`
}

// SpotAreaQuery struct - used to store parameters for query of spots within area.
type SpotAreaQuery struct {
	Latitude  float32 `json:"latitude" validate:"required,number"`
	Longitude float32 `json:"longitude" validate:"required,number"`
	Radius    float32 `json:"radius" validate:"required,number"`
	// Type      string  `json:"type"`
}
