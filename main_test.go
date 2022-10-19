package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDuplicateDomains(t *testing.T) {
	// What do we want to test?
	// We should double-check that none of the results yield DomainCount = 1.
	// The other criteria for the first task have been satisfied in SQL/server code
	var (
		description           = "Duplicate domain endpoint test"
		route                 = "/spots/duplicates"
		expectedCode          = 200
		expectedFirstResultID = "3875c446-0742-4808-8ca6-062aecbade4b"
	)

	// Set up the app as it is done in the main function
	app := Bootstrap()

	req, _ := http.NewRequest(
		"GET",
		route,
		nil,
	)

	// Perform the request plain with the app.
	// The -1 disables request latency.
	res, err := app.Test(req, -1) //nolint:bodyclose

	// verify that no error occured, that is not expected
	assert.Nilf(t, err, "%s: error: %s", description, err)

	// Verify if the status code is as expected
	assert.Equalf(t, expectedCode, res.StatusCode, "%s: status code: %d", description, res.StatusCode)

	// Read the response body using JSON
	spots := Spots{}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&spots)

	assert.Nilf(t, err, description)

	// Check if any of the results have domainCount = 0/1 - should not happen!
	anySingleDomainResults := false
	for _, spot := range spots.Spots {
		if spot.DomainCount == 1 || spot.DomainCount == 0 {
			anySingleDomainResults = true
			break
		}
	}

	assert.Equalf(
		t, expectedCode, res.StatusCode, "%s: any single domain results?: %t", description,
		anySingleDomainResults,
	)

	assert.Equalf(t, expectedFirstResultID, spots.Spots[0].ID, "%s: first result ID: %s", description, spots.Spots[0].ID)
}

// TODO: finish
func TestSpotsInArea(t *testing.T) {
	// What do we want to test?
	// We should double-check that none of the results yield DomainCount = 1.
	// The other criteria for the first task have been satisfied in SQL/server code
	var (
		description           = "Spots in area test"
		route                 = "/spots/inArea"
		expectedCode          = 200
		expectedFirstResultID = "3875c446-0742-4808-8ca6-062aecbade4b"
	)

	var testData = []struct {
		latitude  float32
		longitude float32
		radius    int
		// type string
	}{
		{53.38866, -2.91334, 2000},
		{1, 0, 0},
		{2, -2, -2},
		{0, -1, -1},
		{-1, 0, -1},
	}

	var expectedTestData = []struct {
		latitude  float32
		longitude float32
		radius    int
		// type string
	}{
		{53, -2, 2000},
		{1, 0, 0},
		{2, -2, -2},
		{0, -1, -1},
		{-1, 0, -1},
	}

	// Set up the app as it is done in the main function
	app := Bootstrap()

	req, _ := http.NewRequest(
		"GET",
		route,
		nil,
	)

	// Perform the request plain with the app.
	// The -1 disables request latency.
	res, err := app.Test(req, -1) //nolint:bodyclose

	// verify that no error occured, that is not expected
	assert.Nilf(t, err, "%s: error: %s", description, err)

	// Verify if the status code is as expected
	assert.Equalf(t, expectedCode, res.StatusCode, "%s: status code: %d", description, res.StatusCode)

	// Read the response body using JSON
	spots := Spots{}
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&spots)

	assert.Nilf(t, err, description)

	// Check if any of the results have domainCount = 0/1 - should not happen!
	anySingleDomainResults := false
	for _, spot := range spots.Spots {
		if spot.DomainCount == 1 || spot.DomainCount == 0 {
			anySingleDomainResults = true
			break
		}
	}

	assert.Equalf(
		t, expectedCode, res.StatusCode, "%s: any single domain results?: %t", description,
		anySingleDomainResults,
	)

	assert.Equalf(t, expectedFirstResultID, spots.Spots[0].ID, "%s: first result ID: %s", description, spots.Spots[0].ID)
}
