package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// Each module owns its endpoint, so generated Lua can supply arguments but can
// never choose the host. That is the difference between exposing a capability
// and exposing the network.

// ── ipinfo ───────────────────────────────────────────────────────────────────

// IPInfo exposes the caller's IP geolocation to generated Lua.
type IPInfo struct{}

func (IPInfo) Name() string { return "ipinfo" }

func (IPInfo) Usage() string {
	return `info = layer.ipinfo.get()
-- Looks up the caller's location by IP. Returns a table with:
--   location  (string)  e.g. "Berlin, DE"
--   latitude  (number)
--   longitude (number)`
}

func (IPInfo) Init(l *lua.LState) *lua.LTable {
	t := l.NewTable()
	t.RawSetString("get", l.NewFunction(ipinfoGet))
	return t
}

const ipinfoEndpoint = "https://ipinfo.io/json"

func ipinfoGet(l *lua.LState) int {
	resp, err := httpClient.Get(ipinfoEndpoint)
	if err != nil {
		l.RaiseError("ipinfo: request failed: %v", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		l.RaiseError("ipinfo: read failed: %v", err)
		return 0
	}

	var payload struct {
		City    string `json:"city"`
		Country string `json:"country"`
		Loc     string `json:"loc"` // "lat,lon"
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		l.RaiseError("ipinfo: parse failed: %v", err)
		return 0
	}

	lat, lon, err := parseLoc(payload.Loc)
	if err != nil {
		l.RaiseError("ipinfo: %v", err)
		return 0
	}

	location := payload.City
	if payload.Country != "" {
		location = fmt.Sprintf("%s, %s", payload.City, payload.Country)
	}

	t := l.NewTable()
	t.RawSetString("location", lua.LString(location))
	t.RawSetString("latitude", lua.LNumber(lat))
	t.RawSetString("longitude", lua.LNumber(lon))
	l.Push(t)
	return 1
}

// parseLoc splits ipinfo's "lat,lon" string into two numbers.
func parseLoc(loc string) (float64, float64, error) {
	latStr, lonStr, ok := strings.Cut(loc, ",")
	if !ok {
		return 0, 0, fmt.Errorf("unexpected loc %q", loc)
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("bad latitude in %q", loc)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonStr), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("bad longitude in %q", loc)
	}
	return lat, lon, nil
}

// ── open-meteo ───────────────────────────────────────────────────────────────

// Weather exposes the current weather for a coordinate to generated Lua.
type Weather struct{}

func (Weather) Name() string { return "weather" }

func (Weather) Usage() string {
	return `w = layer.weather.current(latitude, longitude)
-- Fetches the current weather for a coordinate. Returns a table with:
--   temperature (number)  degrees Celsius
--   wind_speed  (number)  km/h
--   time        (string)  observation time`
}

func (Weather) Init(l *lua.LState) *lua.LTable {
	t := l.NewTable()
	t.RawSetString("current", l.NewFunction(weatherCurrent))
	return t
}

const weatherEndpoint = "https://api.open-meteo.com/v1/forecast"

func weatherCurrent(l *lua.LState) int {
	lat := float64(l.CheckNumber(1))
	lon := float64(l.CheckNumber(2))

	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', -1, 64))
	q.Set("current", "temperature_2m,wind_speed_10m")

	resp, err := httpClient.Get(weatherEndpoint + "?" + q.Encode())
	if err != nil {
		l.RaiseError("weather: request failed: %v", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		l.RaiseError("weather: read failed: %v", err)
		return 0
	}

	var payload struct {
		Current struct {
			Time          string  `json:"time"`
			Temperature2m float64 `json:"temperature_2m"`
			WindSpeed10m  float64 `json:"wind_speed_10m"`
		} `json:"current"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		l.RaiseError("weather: parse failed: %v", err)
		return 0
	}

	t := l.NewTable()
	t.RawSetString("temperature", lua.LNumber(payload.Current.Temperature2m))
	t.RawSetString("wind_speed", lua.LNumber(payload.Current.WindSpeed10m))
	t.RawSetString("time", lua.LString(payload.Current.Time))
	l.Push(t)
	return 1
}

// ── shared ───────────────────────────────────────────────────────────────────

// maxResponseBytes caps how much of a response we read, so a misbehaving or
// hostile endpoint cannot blow up the process — the same bounding discipline
// the layer sandbox applies to generated code.
const maxResponseBytes = 1 << 20 // 1 MiB

var httpClient = &http.Client{Timeout: 10 * time.Second}
