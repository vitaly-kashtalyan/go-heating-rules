package main

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/health", health)
	e.GET("/rules", rules)

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

func getRules() (rules Rules, err error) {
	jsonFile, err := os.Open("config/rules.json")
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

type BaseResponse struct {
	Message string `json:"message"`
}

type Rules struct {
	Radiators Circuit `json:"radiators"`
	Floors    Circuit `json:"floors"`
}

type Circuit struct {
	Temperature   float32  `json:"temperature,omitempty"`
	ParentRelayID int      `json:"parent_relay_id,omitempty"`
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
	Time        string  `json:"time"`
	Temperature float32 `json:"temperature"`
}
