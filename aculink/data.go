package aculink

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/pborman/uuid"
)

var pressureKeys = []string{
	"A",
	"B",
	"C",
	"C1",
	"C2",
	"C3",
	"C4",
	"C5",
	"C6",
	"C7",
	"D",
	"PR",
	"TR",
}

var windMapping = map[string]float32{
	"5": 0.0,
	"7": 22.5,
	"3": 45.0,
	"1": 67.5,
	"9": 90.0,
	"B": 112.5,
	"F": 135.0,
	"D": 157.5,
	"C": 180.0,
	"E": 202.5,
	"A": 225.0,
	"8": 247.5,
	"0": 270.0,
	"2": 292.5,
	"6": 315.0,
	"4": 337.5,
}

type JSONTime struct{ time.Time }

func (self JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + self.UTC().Format(time.RFC3339) + `"`), nil
}

func (self *JSONTime) UnmarshalJSON(buf []byte) error {
	return json.Unmarshal(buf, &self.Time)
}

type Data struct {
	UUID          uuid.UUID `json:"uuid"`
	Timestamp     JSONTime  `json:"timestamp"`
	BridgeID      string    `json:"bridge_id"`
	Sensor        string    `json:"sensor"`
	Mt            string    `json:"mt"`
	Battery       *string   `json:"battery,omitempty"`
	SignalRSSI    *int      `json:"signal_rssi,omitempty"`
	TemperatureC  *float32  `json:"temperature_c,omitempty"`
	Humidity      *float32  `json:"humidity,omitempty"`
	WindKMH       *float32  `json:"wind_kmh,omitempty"`
	WindDirection *float32  `json:"wind_direction,omitempty"`
	RainfallMM    *float32  `json:"rainfall_mm,omitempty"`
	PressurePA    *int      `json:"pressure_pa,omitempty"`
}

func (self *Data) JSONString() string {
	b, _ := json.Marshal(self)
	return string(b)
}

func (self *Data) String() string {
	s := fmt.Sprintf(
		"UUID=%s, Timestamp=%s, BridgeID=%s, Sensor=%s, Mt=%s",
		self.UUID,
		self.Timestamp,
		self.BridgeID,
		self.Sensor,
		self.Mt,
	)

	if self.Battery != nil {
		s += fmt.Sprintf(", Battery=%s", *self.Battery)
	}

	if self.SignalRSSI != nil {
		s += fmt.Sprintf(", SignalRSSI=%d", *self.SignalRSSI)
	}

	if self.TemperatureC != nil {
		s += fmt.Sprintf(", TemperatureC=%0.2f", *self.TemperatureC)
	}

	if self.Humidity != nil {
		s += fmt.Sprintf(", Humidity=%0.1f", *self.Humidity)
	}

	if self.WindKMH != nil {
		s += fmt.Sprintf(", WindKMH=%0.2f", *self.WindKMH)
	}

	if self.WindDirection != nil {
		s += fmt.Sprintf(", WindDirection=%0.1f", *self.WindDirection)
	}

	if self.RainfallMM != nil {
		s += fmt.Sprintf(", RainfallMM=%0.3f", *self.RainfallMM)
	}

	if self.PressurePA != nil {
		s += fmt.Sprintf(", PressurePA=%d", *self.PressurePA)
	}

	return s
}

func (self *Data) Parse(s string) error {
	values, err := url.ParseQuery(s)
	if err != nil {
		return err
	}

	mt := values.Get("mt")
	if mt == "" {
		return errors.New("'mt' not found in data")
	}

	bridge_id := values.Get("id")
	if bridge_id == "" {
		return errors.New("'id' not found in data")
	}

	self.BridgeID = bridge_id
	self.Mt = mt

	if mt == "pressure" {
		self.Sensor = "bridge"
		pressure, err := self.getPressure(values)
		if err == nil {
			self.PressurePA = &pressure
		}
		return err
	}

	if sensor := values.Get("sensor"); sensor != "" {
		self.Sensor = sensor
	}

	if temperature := values.Get("temperature"); temperature != "" {
		t, err := self.getTemperature(temperature)
		if err != nil {
			return err
		}
		self.TemperatureC = &t
	}

	if humidity := values.Get("humidity"); humidity != "" {
		h, err := self.getHumidity(humidity)
		if err != nil {
			return err
		}
		self.Humidity = &h
	}

	if rainfall := values.Get("rainfall"); rainfall != "" {
		r, err := self.getRainfall(rainfall)
		if err != nil {
			return err
		}
		self.RainfallMM = &r
	}

	if windspeed := values.Get("windspeed"); windspeed != "" {
		w, err := self.getWindspeed(windspeed)
		if err != nil {
			return err
		}
		self.WindKMH = &w
	}

	if winddir := values.Get("winddir"); winddir != "" {
		w, err := self.getWindDirection(winddir)
		if err != nil {
			return err
		}
		self.WindDirection = &w
	}

	if battery := values.Get("battery"); battery != "" {
		self.Battery = &battery
	}

	if rssi := values.Get("rssi"); rssi != "" {
		r, err := self.getRSSI(rssi)
		if err != nil {
			return err
		}
		self.SignalRSSI = &r
	}

	return nil
}

func (self *Data) getTemperature(temperature string) (float32, error) {
	// temperature == AXYYZZZZZZ = Y.Z Celsius -- If X is '-' then negative
	// Only care about 2 digits after decimal. Grab 3 so we can round
	tmpi, err := strconv.ParseInt(temperature[1:len(temperature)-3], 10, 0)
	if err != nil {
		return 0, err
	}
	return float32((5+tmpi)/10) / 100.0, nil
}

func (self *Data) getHumidity(humidity string) (float32, error) {
	// humidity == AXXXY == X.Y%
	tmpi, err := strconv.ParseInt(humidity[1:len(humidity)], 10, 0)
	if err != nil {
		return 0, err
	}

	return float32(tmpi) / 10, nil
}

func (self *Data) getRainfall(rainfall string) (float32, error) {
	// rainfall == AXXXXYYY == X.Y mm
	tmpi, err := strconv.ParseInt(rainfall[1:len(rainfall)], 10, 0)
	if err != nil {
		return 0, err
	}

	return float32(tmpi) / 1000, nil
}

func (self *Data) getWindspeed(windspeed string) (float32, error) {
	// windspeed == AXXXXXXYYY == X.Y mm/s
	// Ignore last 3 chars -- more digits than we need. Converting to
	// km/h would be X * 3600 / 1000000 or X * 36 / 10000
	// 2 digits after decimal place is enough, so we'll
	// use float32((50 + X * 36) / 100) / 100.0 (+50 is to round)

	tmpi, err := strconv.ParseInt(windspeed[1:len(windspeed)-3], 10, 0)
	if err != nil {
		return 0, err
	}
	return float32((50+tmpi*36)/100) / 100.0, nil
}

func (self *Data) getWindDirection(winddir string) (float32, error) {
	v, ok := windMapping[winddir]
	if !ok {
		return 0, errors.New("Unknown wind direction found")
	}
	return v, nil
}

func (self *Data) getRSSI(rssi string) (int, error) {
	// 0 to 4 -- We'll turn into %
	tmpi, err := strconv.ParseInt(rssi, 10, 0)
	if err != nil {
		return 0, err
	}
	return int(tmpi) * 25, nil
}

func (self *Data) getPressure(values url.Values) (int, error) {
	v := map[string]float64{}

	for _, k := range pressureKeys {
		tmpv := values.Get(k)
		if tmpv == "" {
			return 0, errors.New(
				fmt.Sprintf("No value for '%s' found when parsing pressure", k),
			)
		}
		uintv, err := strconv.ParseInt(tmpv, 16, 64)
		if err != nil {
			return 0, err
		}
		v[k] = float64(uintv)
	}

	var coef float64
	if v["TR"] >= v["C5"] {
		coef = v["A"]
	} else {
		coef = v["B"]
	}

	dUTpart := (v["TR"] - v["C5"]) / 128
	dUT := v["TR"] - v["C5"] - (dUTpart * dUTpart * coef / math.Pow(2, v["C"]))
	OFF := (v["C2"] + (v["C4"]-1024)*dUT/16384) * 4
	SENS := v["C1"] + v["C3"]*dUT/1024
	X := SENS*(v["PR"]-7168)/16384 - OFF
	P := (X * 100 / 32) + (v["C7"] * 10)
	// T := 250 + (dUT * v["C6"] / 65536) - dUT/math.pow(2, v["D"])

	return int(P), nil
}

func NewData(data ...string) (*Data, error) {
	if len(data) > 1 {
		panic("NewData supports 1 argument max")
	}
	d := &Data{
		UUID:      uuid.NewRandom(),
		Timestamp: JSONTime{time.Now()},
	}
	if len(data) > 0 {
		err := d.Parse(data[0])
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

func DataFromJSON(s string) (*Data, error) {
	d := &Data{}
	if err := json.Unmarshal([]byte(s), d); err != nil {
		return nil, err
	}
	return d, nil
}
