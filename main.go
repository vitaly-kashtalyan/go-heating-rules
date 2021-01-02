package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const PathRulesFile = "config/rules.json"

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/health", health)
	e.GET("/rules", rules)
	e.GET("/sensors", sensors)
	e.PATCH("/relays", relays)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

func health(c echo.Context) error {
	return c.JSON(http.StatusOK, BaseResponse{
		Message: http.StatusText(http.StatusOK),
	})
}

func rules(c echo.Context) error {
	rules, err := getRules()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, BaseResponse{
			Message: err.Error(),
		})
	} else {
		return c.JSON(http.StatusOK, rules)
	}
}

func sensors(c echo.Context) error {
	status, err := getTemperature()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, BaseResponse{
			Message: err.Error(),
		})
	} else {
		return c.JSON(http.StatusOK, status)
	}
}

func relays(c echo.Context) error {
	err := updateRelay(c)
	return prepareResponse(c, err)
}

func prepareResponse(c echo.Context, err error) error {
	if err != nil {
		code := http.StatusInternalServerError
		if err.Error() == "Not Found" {
			code = http.StatusNotFound
		}

		return c.JSON(code, BaseResponse{
			Message: err.Error(),
		})
	}
	return c.JSON(http.StatusNoContent, nil)
}

func updateRelay(c echo.Context) (err error) {
	isNotFound := true
	rules, err := getRules()
	if err != nil {
		return
	}
	var jsonBody relayPatch
	err = c.Bind(&jsonBody)
	if err != nil {
		return
	}
	err = validateSchedule(jsonBody.Schedule)
	if err != nil {
		return
	}
	for ci, circuit := range rules.Circuits {
		for i, _ := range circuit.Relays {
			if jsonBody.Pin == rules.Circuits[ci].Relays[i].Pin && jsonBody.Dec == rules.Circuits[ci].Relays[i].Dec {
				if jsonBody.Name != "" {
					rules.Circuits[ci].Relays[i].Name = jsonBody.Name
				}
				if jsonBody.Enable != nil {
					rules.Circuits[ci].Relays[i].Enable = *jsonBody.Enable
				}
				if len(jsonBody.Schedule) == 0 {
					rules.Circuits[ci].Relays[i].Schedule = []Schedule{}
				} else {
					rules.Circuits[ci].Relays[i].Schedule = jsonBody.Schedule
				}
				isNotFound = false
			}
		}
	}
	if isNotFound {
		return fmt.Errorf("Not Found")
	}
	return writeObjectToJson(rules)
}

func validateSchedule(schedules []Schedule) (err error) {
	if len(schedules) != 0 {
		for _, s := range schedules {
			if s.Time == "" {
				j, _ := json.Marshal(schedules)
				return fmt.Errorf("time field must not be empty:", string(j))
			}
		}
	}
	return
}

func getRules() (rules Rules, err error) {
	jsonFile, err := os.Open(PathRulesFile)
	if err != nil {
		return
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(byteValue, &rules)
	return
}

func getTemperature() (status Status, err error) {
	rules, err := getRules()
	if err != nil {
		return
	}
	status.ShortRelays = appendSensorsData(rules)
	return
}

func appendSensorsData(rules Rules) (s []Sensor) {
	for _, circuit := range rules.Circuits {
		for _, relay := range circuit.Relays {
			s = append(s, Sensor{
				Pin:         relay.Pin,
				Dec:         relay.Dec,
				Enable:      relay.Enable,
				RelayId:     relay.RelayId,
				Temperature: getTemperatureBySchedule(relay.Schedule, circuit.Temperature),
			})
		}
	}
	return
}

func getTemperatureBySchedule(s []Schedule, t float32) (temp float32) {
	now := time.Now().In(time.FixedZone("UTC+3", 3*60*60))
	prevDiff := 999999.9
	temp = t

	if len(s) != 0 {
		for _, h := range s {
			prepareTime := fmt.Sprintf("%v-%v-%v %v +0300", now.Year(), now.Month().String(), now.Day(), h.Time)
			timeT, err := time.Parse("2006-January-2 15:04 PM -0700", prepareTime)
			if err == nil {
				diff := now.Sub(timeT).Seconds()
				if diff > 0 && prevDiff > diff {
					temp = h.Temperature
					prevDiff = diff
				}
			}
		}
	}
	return
}

func writeObjectToJson(data interface{}) (err error) {
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return
	}
	return ioutil.WriteFile(PathRulesFile, file, 0644)
}

type BaseResponse struct {
	Message string `json:"message"`
}

type Rules struct {
	Circuits []Circuit `json:"circuits"`
}

type Circuit struct {
	Name          string   `json:"name"`
	Temperature   float32  `json:"temperature"`
	ParentRelayID int      `json:"parent_relay_id"`
	Relays        []Relays `json:"relays"`
}

type Relays struct {
	Pin      int        `json:"pin"`
	Dec      string     `json:"dec"`
	RelayId  int        `json:"relay_id"`
	Name     string     `json:"name"`
	Enable   bool       `json:"enable"`
	Schedule []Schedule `json:"schedule"`
}

type Schedule struct {
	Time        string  `json:"time" binding:"required"`
	Temperature float32 `json:"temperature" binding:"required"`
}

type Status struct {
	ShortRelays []Sensor `json:"sensors"`
}

type Sensor struct {
	Pin         int     `json:"pin"`
	Dec         string  `json:"dec"`
	RelayId     int     `json:"relay_id"`
	Temperature float32 `json:"temperature"`
	Enable      bool    `json:"enable"`
}

type relayPatch struct {
	Pin      int        `json:"pin" binding:"required"`
	Dec      string     `json:"dec"`
	Name     string     `json:"name,omitempty"`
	Enable   *bool      `json:"enable"`
	Schedule []Schedule `json:"schedule"`
}
