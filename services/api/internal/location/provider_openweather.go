package location

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	openWeatherCurrentURL        = "https://api.openweathermap.org/data/2.5/weather"
	openWeatherHourlyTimelineURL = "https://api.openweathermap.org/data/4.0/onecall/timeline/1h"
	openWeatherTimelineURL       = "https://api.openweathermap.org/data/4.0/onecall/timeline/1day"

	openWeatherTimeout       = 8 * time.Second
	openWeatherAgent         = "trip-app/1.0"
	openWeatherPerPage       = 10
	openWeatherPagesMax      = 3 // 3 × 10 = 30 days
	openWeatherDaysMax       = openWeatherPerPage * openWeatherPagesMax
	openWeatherDayEpoch      = int64(86_400)
	openWeatherHourlyHorizon = 24 * time.Hour
)

type OpenWeatherProvider struct {
	apiKey string
	client *http.Client
	logger *zap.Logger
}

func NewOpenWeatherProvider(apiKey string, logger *zap.Logger) *OpenWeatherProvider {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OpenWeatherProvider{
		apiKey: strings.TrimSpace(apiKey),
		client: &http.Client{Timeout: openWeatherTimeout},
		logger: logger.With(zap.String("layer", "location.openweather")),
	}
}

func (o *OpenWeatherProvider) available() bool { return o != nil && o.apiKey != "" }

// ---- raw response shapes ----

type owMain struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Humidity  int     `json:"humidity"`
}

type owWeather struct {
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type owWind struct {
	Speed float64 `json:"speed"`
}

type owCurrentResp struct {
	Dt       int64       `json:"dt"`
	Main     owMain      `json:"main"`
	Weather  []owWeather `json:"weather"`
	Wind     owWind      `json:"wind"`
	Timezone int         `json:"timezone"`
}

type owHourlyTimelineEntry struct {
	Dt        int64       `json:"dt"`
	Temp      float64     `json:"temp"`
	FeelsLike float64     `json:"feels_like"`
	Humidity  int         `json:"humidity"`
	WindSpeed float64     `json:"wind_speed"`
	Weather   []owWeather `json:"weather"`
	Pop       float64     `json:"pop"`
}

type owHourlyTimelineResp struct {
	Lat            float64                 `json:"lat"`
	Lon            float64                 `json:"lon"`
	Timezone       string                  `json:"timezone"`
	TimezoneOffset int                     `json:"timezone_offset"`
	Data           []owHourlyTimelineEntry `json:"data"`
}

// One Call 4.0 daily timeline shapes.

type owTempBlock struct {
	Day   float64 `json:"day"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Night float64 `json:"night"`
	Eve   float64 `json:"eve"`
	Morn  float64 `json:"morn"`
}

type owTimelineEntry struct {
	Dt      int64       `json:"dt"`
	Temp    owTempBlock `json:"temp"`
	Weather []owWeather `json:"weather"`
	Pop     float64     `json:"pop"`
}

type owTimelineResp struct {
	Lat            float64           `json:"lat"`
	Lon            float64           `json:"lon"`
	Timezone       string            `json:"timezone"`
	TimezoneOffset int               `json:"timezone_offset"`
	Data           []owTimelineEntry `json:"data"`
	Next           string            `json:"next"`
	Prev           string            `json:"prev"`
}

func (o *OpenWeatherProvider) Forecast(ctx context.Context, at LatLng, lang, units string) (*WeatherForecast, error) {
	if !o.available() {
		return nil, ErrProviderUnavailable
	}
	unitSystem := "metric"
	if strings.EqualFold(units, "imperial") {
		unitSystem = "imperial"
	}

	current, err := o.fetchCurrent(ctx, at, unitSystem, lang)
	if err != nil {
		return nil, err
	}

	daily, tz, err := o.fetchDailyTimeline(ctx, at, unitSystem, lang)
	if err != nil {
		return nil, err
	}
	if tz == nil {
		tz = time.FixedZone("owm", current.Timezone)
	}

	hourly, err := o.fetchHourlyTimeline(ctx, at, unitSystem, lang, tz)
	if err != nil {
		// Hourly is best-effort — surface a warning but don't fail the whole call.
		o.logger.Warn("openweather hourly forecast failed", zap.Error(err))
		hourly = nil
	}

	out := &WeatherForecast{
		Location: at,
		Timezone: tz.String(),
		Current: &WeatherPoint{
			Time:     time.Unix(current.Dt, 0).UTC(),
			TempC:    normalizeTemp(current.Main.Temp, unitSystem),
			FeelsLC:  normalizeTemp(current.Main.FeelsLike, unitSystem),
			Humidity: current.Main.Humidity,
			WindKph:  windToKph(current.Wind.Speed, unitSystem),
			Icon:     owIconURL(firstWeather(current.Weather).Icon),
			Summary:  firstWeather(current.Weather).Description,
		},
		Hourly: hourly,
		Daily:  daily,
	}
	return out, nil
}

// One Call 4.0 `/timeline/1h` — true 1-hour granularity. Free-tier accounts
// need to be subscribed to One Call by Call for this to succeed; upstream will
// 401 otherwise and we log-and-continue.
func (o *OpenWeatherProvider) fetchHourlyTimeline(ctx context.Context, at LatLng, units, lang string, tz *time.Location) ([]WeatherPoint, error) {
	q := o.baseQuery(at, units, lang)
	var resp owHourlyTimelineResp
	if err := o.getJSON(ctx, openWeatherHourlyTimelineURL+"?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	if tz == nil {
		tz = time.FixedZone(resp.Timezone, resp.TimezoneOffset)
	}
	now := time.Now()
	horizon := now.Add(openWeatherHourlyHorizon)
	minTime := now.Add(-1 * time.Hour)
	out := make([]WeatherPoint, 0, len(resp.Data))
	for _, e := range resp.Data {
		ts := time.Unix(e.Dt, 0)
		if ts.Before(minTime) || ts.After(horizon) {
			continue
		}
		w := firstWeather(e.Weather)
		out = append(out, WeatherPoint{
			Time:     ts.In(tz),
			TempC:    normalizeTemp(e.Temp, units),
			FeelsLC:  normalizeTemp(e.FeelsLike, units),
			Humidity: e.Humidity,
			WindKph:  windToKph(e.WindSpeed, units),
			Icon:     owIconURL(w.Icon),
			Summary:  w.Description,
			PopPct:   e.Pop * 100,
		})
	}
	return out, nil
}

func (o *OpenWeatherProvider) fetchCurrent(ctx context.Context, at LatLng, units, lang string) (*owCurrentResp, error) {
	q := o.baseQuery(at, units, lang)
	var resp owCurrentResp
	if err := o.getJSON(ctx, openWeatherCurrentURL+"?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (o *OpenWeatherProvider) fetchDailyTimeline(ctx context.Context, at LatLng, units, lang string) ([]WeatherDay, *time.Location, error) {
	base := o.baseQuery(at, units, lang)
	base.Set("cnt", strconv.Itoa(openWeatherPerPage))

	var (
		out   = make([]WeatherDay, 0, openWeatherDaysMax)
		tz    *time.Location
		start int64
	)

	for page := 0; page < openWeatherPagesMax; page++ {
		q := cloneValues(base)
		if start > 0 {
			q.Set("start", strconv.FormatInt(start, 10))
		}

		var resp owTimelineResp
		if err := o.getJSON(ctx, openWeatherTimelineURL+"?"+q.Encode(), &resp); err != nil {
			if page == 0 {
				return nil, nil, err
			}
			o.logger.Warn("openweather timeline partial",
				zap.Int("page", page), zap.Error(err))
			break
		}
		if len(resp.Data) == 0 {
			break
		}
		if tz == nil {
			tz = time.FixedZone(resp.Timezone, resp.TimezoneOffset)
		}
		for _, e := range resp.Data {
			local := time.Unix(e.Dt, 0).In(tz)
			w := firstWeather(e.Weather)
			out = append(out, WeatherDay{
				Date:     time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, tz),
				TempMinC: normalizeTemp(e.Temp.Min, units),
				TempMaxC: normalizeTemp(e.Temp.Max, units),
				Icon:     owIconURL(w.Icon),
				Summary:  w.Description,
				PopPct:   e.Pop * 100,
			})
		}
		start = resp.Data[len(resp.Data)-1].Dt + openWeatherDayEpoch
	}

	return out, tz, nil
}

func (o *OpenWeatherProvider) baseQuery(at LatLng, units, lang string) url.Values {
	q := url.Values{}
	q.Set("lat", fmt.Sprintf("%f", at.Lat))
	q.Set("lon", fmt.Sprintf("%f", at.Lng))
	q.Set("appid", o.apiKey)
	q.Set("units", units)
	if lang != "" {
		q.Set("lang", lang)
	}
	return q
}

func (o *OpenWeatherProvider) getJSON(ctx context.Context, fullURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", openWeatherAgent)
	res, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("openweather: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		msg := extractOWError(res.StatusCode, raw)
		switch res.StatusCode {
		case http.StatusNotFound:
			// No coverage for the given coords.
			return fmt.Errorf("openweather %s: %w", msg, ErrProviderNoResult)
		case http.StatusUnauthorized, http.StatusForbidden:
			// Invalid / missing API key — operational config issue, not a code bug.
			o.logger.Warn("openweather auth failed", zap.Int("status", res.StatusCode), zap.String("body", string(raw)))
			return fmt.Errorf("openweather %s: %w", msg, ErrProviderUnavailable)
		}
		return fmt.Errorf("openweather: %s", msg)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func normalizeTemp(v float64, units string) float64 {
	if strings.EqualFold(units, "imperial") {
		return (v - 32) * 5 / 9
	}
	return v
}

// OpenWeather reports wind as m/s under metric, mph under imperial.
func windToKph(v float64, units string) float64 {
	if strings.EqualFold(units, "imperial") {
		return v * 1.609344
	}
	return v * 3.6
}

func firstWeather(ws []owWeather) owWeather {
	if len(ws) == 0 {
		return owWeather{}
	}
	return ws[0]
}

func owIconURL(icon string) string {
	if icon == "" {
		return ""
	}
	return "https://openweathermap.org/img/wn/" + icon + "@2x.png"
}

func extractOWError(status int, raw []byte) string {
	var envelope struct {
		Cod     any    `json:"cod"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Message != "" {
		return fmt.Sprintf("http %d: %s", status, envelope.Message)
	}
	return fmt.Sprintf("http %d: %s", status, string(raw))
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}
