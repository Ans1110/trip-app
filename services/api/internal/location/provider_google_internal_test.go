package location

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleSearchResp = `{
  "places": [{
    "id": "ChIJ1",
    "displayName": {"text": "Cafe One"},
    "location": {"latitude": 25.04, "longitude": 121.53},
    "photos": [{"name": "places/ChIJ1/photos/AAA"}, {"name": "places/ChIJ1/photos/BBB"}],
    "utcOffsetMinutes": 480,
    "regularOpeningHours": {
      "periods": [
        {"open": {"day": 1, "hour": 8, "minute": 0}, "close": {"day": 1, "hour": 17, "minute": 30}},
        {"open": {"day": 0, "hour": 0, "minute": 0}}
      ]
    }
  }, {
    "id": "ChIJ2",
    "displayName": {"text": "No Extras"},
    "location": {"latitude": 1, "longitude": 2}
  }]
}`

func TestMapNewPlaces_PhotoAndHours(t *testing.T) {
	var resp googleSearchResp
	require.NoError(t, json.Unmarshal([]byte(sampleSearchResp), &resp))

	out := mapNewPlaces(resp.Places)
	require.Len(t, out, 2)

	assert.Equal(t, "places/ChIJ1/photos/AAA", out[0].Photo)
	require.NotNil(t, out[0].Hours)
	assert.Equal(t, 480, out[0].Hours.UTCOffsetMinutes)
	require.Len(t, out[0].Hours.Periods, 2)
	assert.Equal(t, HoursPoint{Day: 1, Hour: 8}, out[0].Hours.Periods[0].Open)
	assert.Equal(t, &HoursPoint{Day: 1, Hour: 17, Minute: 30}, out[0].Hours.Periods[0].Close)
	assert.Nil(t, out[0].Hours.Periods[1].Close)

	assert.Empty(t, out[1].Photo)
	assert.Nil(t, out[1].Hours)
}
