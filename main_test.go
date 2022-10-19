package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
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
		expectedTotal         = 248
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

	defer res.Body.Close()

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
	assert.Equalf(t, expectedTotal, spots.Total, "%s: total count: %d", description, spots.Total)
}

//nolint:funlen
func TestSpotsInArea(t *testing.T) {
	// What do we want to test?
	// We should test that the endpoint actually works.
	// It should report back the correct total/last result ID, that way we know that it's still sorting correctly.
	var (
		description  = "Spots in area test"
		route        = "/spots/inArea"
		expectedCode = 200
	)

	// Uses table-driven testing to make things 100x easier.
	var testData = []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Radius    int     `json:"radius"`
		// type string
	}{
		{53.38866, -2.91334, 1000},
		{53.38866, -2.91334, 2000},
		{53.38866, -2.91334, 5000},
		{53.2, -2.92, 5000},
	}

	var expectedTestResults = []struct {
		total        int
		lastResultID string
	}{
		{7, "d3b998c5-6261-4657-8a11-cab10330b2ec"},
		{12, "2f152f74-dfd1-47d8-8c24-1f3781bc6292"},
		{42, "4160ba38-4d54-4734-be76-e9cc9d907d8b"},
		{10, "1dd7244a-a9a8-435d-9e17-77cba282645b"},
	}

	// Set up the app as it is done in the main function
	app := Bootstrap()

	for i, test := range testData {
		payloadBuf := new(bytes.Buffer)
		err := json.NewEncoder(payloadBuf).Encode(test)

		t.Log(payloadBuf.String())

		assert.Nilf(t, err, "%s: error: %s", description, err)

		req, _ := http.NewRequest(
			"GET",
			route,
			payloadBuf,
		)

		req.Header.Set("Content-Type", "application/json; charset=UTF-8")

		// Perform the request plain with the app.
		// The -1 disables request latency.
		res, err := app.Test(req, -1) //nolint:bodyclose
		assert.Nilf(t, err, "%s: error: %s", description, err)

		defer res.Body.Close()

		// Verify if the status code is as expected
		assert.Equalf(t, expectedCode, res.StatusCode, "%s: status code: %d", description, res.StatusCode)

		// Read the response body using JSON
		spots := Spots{}

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&spots)
		if err != nil {
			assert.Nilf(t, err, "%s: error: %s", description, err)
			return
		}

		assert.Equalf(
			t, spots.Total, expectedTestResults[i].total, "%s: actual total results: %d", description,
			spots.Total,
		)

		assert.Equalf(
			t, spots.Spots[spots.Total-1].ID, expectedTestResults[i].lastResultID, "%s: last result ID: %s", description,
			spots.Spots[spots.Total-1].ID,
		)
	}
}

func TestSpotsInAreaValidation(t *testing.T) {
	// What do we want to test?
	// That the endpoint doesn't just accept anything.
	var (
		description  = "Spots in area test"
		route        = "/spots/inArea"
		expectedCode = fiber.StatusBadRequest
	)

	// Uses table-driven testing to make things 100x easier.
	var testData = []struct {
		Latitude  any `json:"latitude"`
		Longitude any `json:"longitude"`
		Radius    any `json:"radius"`
		// type string
	}{
		{0, 0, 0},
		{"abcd", "bcde", "achadf"},
	}

	// Set up the app as it is done in the main function
	app := Bootstrap()

	for _, test := range testData {
		payloadBuf := new(bytes.Buffer)
		err := json.NewEncoder(payloadBuf).Encode(test)

		t.Log(payloadBuf.String())
		assert.Nilf(t, err, "%s: error: %s", description, err)

		req, _ := http.NewRequest(
			"GET",
			route,
			payloadBuf,
		)

		req.Header.Set("Content-Type", "application/json; charset=UTF-8")

		// Perform the request plain with the app.
		// The -1 disables request latency.
		res, err := app.Test(req, -1) //nolint:bodyclose
		assert.Nilf(t, err, "%s: error: %s", description, err)

		defer res.Body.Close()

		// Verify if the status code is as expected
		assert.Equalf(t, expectedCode, res.StatusCode, "%s: status code: %d", description, res.StatusCode)

		// Read the response body using JSON
		var errorMsg interface{}

		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&errorMsg)
		if err != nil {
			assert.Nilf(t, err, "%s: error: %s", description, err)
			return
		}

		t.Logf("%+v\n", errorMsg)

		// now it should return this
		/*

			[
			  {
			    "FailedField": "SpotAreaQuery.Latitude",
			    "Tag": "required",
			    "Value": ""
			  },
			  {
			    "FailedField": "SpotAreaQuery.Longitude",
			    "Tag": "required",
			    "Value": ""
			  },
			  {
			    "FailedField": "SpotAreaQuery.Radius",
			    "Tag": "required",
			    "Value": ""
			  }
			]

		*/
	}
}
