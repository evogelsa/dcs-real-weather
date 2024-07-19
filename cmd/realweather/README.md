# DCS Real Weather

## Usage

1) Create an account at [checkwx](https://checkwxapi.com/).
2) Find your API key from your account details and copy it.
3) Download the
[latest release](https://github.com/evogelsa/DCS-real-weather/releases/latest).
4) Extract the files in the release zip.
5) Open the provided `config.json` with a text editor of choice.
6) Add your API key between the quotes and configure the other settings to your
liking. A description of each of the settings is provided
[below](#config-file).
7) Save your changes and ensure the config file remains inside the same
directory that the realweather.exe file is located.
8) Create or configure the mission file you want to be updated with the real
weather.
9) To run the utility you may either manually run the realweather.exe, or you
can use a script to automate the process. Some examples are provided in
[examples](/examples).
10) Enjoy automatic weather updates to your mission!

Alternatively if you prefer a more feature-full and comprehensive DCS server
experience, check out [DCS Server
Bot](https://github.com/Special-K-s-Flightsim-Bots/DCSServerBot). Real Weather
is a supported extension and can be integrated directly with the tool.

> [!NOTE]
> Generally your input file should be different from your output file. This will
> leave your main mission template untouched.

## Config file

An example configuration file can be found below along with an explanation of
each parameter.

### Example config file

```json
{
  "api-key": "", // your api key from checkwx
  "files": {
    "input-mission": "mission.miz",      // path of mission file to be updated
    "output-mission": "realweather.miz", // path of output mission file
    "log": "logfile.log"                 // path of log file, "" disables
  },
  "metar": {
    "icao": "KDLH",          // ICAO of the aiport to fetch METAR from
    "runway-elevation": 0,   // elevation of runway in meters MSL
    "remarks": "",           // addtional remarks for METAR, customization only
    "add-to-brief": true,    // add METAR text to bottom of mission brief
    "use-custom-data": false // use custom data from checkwx.json file
  },
  "options": {
    "update-time": true,     // set to false to disable time being updated
    "update-weather": true,  // set to false to disable weather being updated
    "time-offset": "-5h30m", // time offset from system time
    "wind": {
      "minimum": 0,            // min allowed wind speed in m/s, at least 0
      "maximum": 50,           // max allowed wind speed in m/s, at most 50
      "gust-minimum": 0,       // min allowed gust speed in m/s, at least 0
      "gust-maximum": 50,      // max allowed gust speed in m/s, at most 50
      "open-meteo": true,      // use Open Meteo API for winds aloft
      "stability": 0.143,      // atmospheric stability for wind calculations
      "fixed-reference": false // use a fixed reference for wind calculations
    },
    "clouds": {
      "disallowed-presets": [        // list of presets you do not want selected
          "RainyPreset1",
          "RainyPreset2",
          "RainyPreset3"
          ],
      "fallback-to-no-preset": true, // use custom wx if no preset match found
      "default-preset": "Preset7"    // default preset to use if no match found
    },
    "fog": {
      "enabled": true,           // set to false to disable fog
      "thickness-minimum": 0,    // min thickness of fog in meters, at least 0
      "thickness-maximum": 1000, // max thickness of fog in meters, at most 1000
      "visibility-minimum": 0,   // min vis through fog in meters, at least 0
      "visibility-maximum": 6000 // max vis through fog in meters, at most 6000
    },
    "dust": {
      "enabled": true,           // set to false to disable dust and smoke
      "visibility-minimum": 300, // min vis through dust in meters, at least 300
      "visibility-maximum": 3000 // max vis through dust in meters, at most 3000
    }
  }
}
```

### Config Parameters

* `api-key`: string
  * Your API key from checkwx, should be a mix of letters and numbers.
* `files`: string
  * `input-mission`: string
    * This is the path of the mission file that you want to apply real weather
      to. This is typically a relative path from the Real Weather executable,
      but it can be an absolute path too.
  * `output-mission`: string
    * This is the path that you want the modified mission to be output to. This
      should generally be different from your input file.
  * `log`: string
    * This is the path that you want Real Weather to create its log at. If you
      do not want logging, you can disable with "".
* `metar`
  * `icao`: string
    * This is the ICAO of the airport you would like weather to be pulled from.
  * `runway-elevation`: integer
    * The runway/airport elevation of the ICAO configured in meters. This is
      used to convert cloud heights between MSL and AGL values for METAR
      accuracy. Using this value the METAR will report the clouds in hundreds of
      feet AGL. Additionally the elevation is used to properly represent the QNH
      in DCS. This value will also be used for the reference height in wind
      calculation if `open-meteo` and `fixed-reference` are false.
  * `remarks`: string
    * This adds a RMK section in the METAR string. There is not functional
      impact of this setting. It's used for customization only.
  * `add-to-brief`: boolean
    * If true, Real Weather will add the generated METAR to your mission brief.
      If your brief contains `==Real Weather METAR==` anywhere in the brief,
      then the METAR will be inserted at the line immediately following,
      replacing whatever is currently there. Otherwise, the METAR will be
      appended to the end of your mission brief.
  * `use-custom-data`: boolean
    * If true, Real Weather will load METAR data from a checkwx.json file
      and overwrite data from the CheckWX API. This data must be in the same
      format CheckWX would provide. See
      [examples/checkwx.json](examples/checkwx.json) and
      [checkwxapi.com](https://www.checkwxapi.com/documentation/metar) for more
      info. Only the parameters shown in the example are currently supported.
      If not all parameters are given, the missing data will come from CheckWX.
* `options`
  * `update-time`: boolean
    * Disable/enable Real Weather modifying your mission time.
  * `update-weather`: boolean
    * Disable/enable Real Weather modifying your mission weather.
  * `time-offset`: string
    * This is the offset from *system time* used when updating the mission time.
      This value should be a string such as "1.5h" or "-2h45m". Supported units
      are nanoseconds "ns", microseconds "us", milliseconds "ms", seconds "s",
      minutes "m", and hours "h".
  * `wind`
    * `minimum`: integer
      * This is the minimum wind speed in meters per second that Real Weather
        will apply to your mission. This value must be at least 0.
    * `maximum`: integer
      * This is the maximum wind speed in meters per second that Real Weather
        will apply to your mission. This value must be at most 50.
    * `gust-minimum`: integer
      * This is the minimum gust speed in meters per second that Real Weather
        will apply to your mission. This value must be at least 0.
    * `gust-maximum`: integer
      * This is the maximum gust speed in meters per second that Real Weather
        will apply to your mission. This value must be at most 50.
    * `open-meteo`: boolean
      * If true, Real Weather will use the Open Meteo API to get winds aloft
        data. If false, Real Weather will use the wind profile power law to
        estimate winds aloft.
    * `stability`: float
      * This is the atmospheric stability number used in the wind profile power
        law. This is used when calculating wind speeds at altitudes other than
        the airport elevation. 0.143 is generally a good setting for this, but
        it can be configured to your liking. Larger values generally equate to
        less stable atmosphere with bigger difference between ground winds and
        winds aloft, and smaller values generally equate to more stable
        atmospheres with ground winds being closer to winds aloft. See the
        additional notes below for more information. This parameters only does
        something if `open-meteo` is false or if the Open Meteo API is
        unavailable.
    * `fixed-reference`: boolean
      * Disable/enable using a fixed reference point when calculating winds
        aloft. If false, Real Weather will use the `runway-elevation` as the
        wind reference point for calculting winds aloft. If true, Real Weather
        will use 1 meter MSL as as the reference height for wind calculations.
        Generally this should be set to false. This parameter only does
        something if `open-meteo` is false or if the Open Meteo API is
        unavailable.
  * `clouds`
    * `disallowed-presets`: string array
      * This is a list of all the presets you do not want to be chosen. This
        can be an empty list [] if you do not want to disable any presets.
        Available preset options can be seen in the additional notes below.
    * `fallback-to-no-preset`: boolean
      * If this is true, Real Weather will use the legacy weather (no preset)
        when a suitable weather preset is not found.
    * `default-preset`: string
      * This is the default preset that will be used if `fallback-to-no-preset`
        is false and no suitable preset can be found in the allowed presets.
        Leave this as "" to disable and use clear skies as the default.
  * `fog`
    * `enabled`: boolean
      * Disable/enable fog when apply weather to your mission.
    * `thickness-minimum`: integer
      * This is the minimum fog thickness in meters that will be used when
        applying fog to your mission. This must be at least 0 and less than
        `thickness-maximum`.
    * `thickness-maximum`: integer
      * This is the maximum fog thickness in meters that will be used when
        applying fog to your mission. This must be at most 1000 and greater than
        `thickness-minimum`.
    * `visibility-minimum`: integer
      * This is the minimum visibility in meters that will be used when
        applying fog to your mission. This must be at least 0 and less than
        `visibility-maximum`.
    * `visibility-maximum`: integer
      * This is the maximum visibility in meters that will be used when
        applying fog to your mission. This must be at most 6000 and greater than
        `visibility-minimum`.
  * `dust`
    * `enabled`: boolean
      * Disable/enable dust when applying weather to your mission.
    * `visibility-minimum`: integer
      * This is the minimum visibility in meters that will be used when
        applying dust to your mission. This must be at least 300 and less than
        `visibility-maximum`.
    * `visibility-maximum`: integer
      * This is the maximum visibility in meters that will be used when
        applying dust to your mission. This must be at most 3000 and greater
        than `visibility-minimum`.

> [!TIP]
> For more info on stability, see these links: [1][1], [2][2], [3][3].

> [!NOTE]
> Fog thickness is not reported by a METAR, so the thickness in DCS will be a
> randomly chosen value between your configured min and max.

### Preset table

The DCS presets are shown in following table. These preset names are the same
names that can be added to the `disallowed-presets` and `default-preset` config
parameters.

| Preset Name    | Cloud Layers         |
|----------------|----------------------|
| "Preset1"      | FEW070               |
| "Preset2"      | FEW080 SCT230        |
| "Preset3"      | SCT080 FEW210        |
| "Preset4"      | SCT080 SCT240        |
| "Preset5"      | SCT140 FEW270 BKN400 |
| "Preset6"      | SCT080 FEW400        |
| "Preset7"      | BKN075 SCT210 SCT400 |
| "Preset8"      | SCT180 FEW360 FEW400 |
| "Preset9"      | BKN075 SCT200 FEW410 |
| "Preset10"     | SCT180 FEW360 FEW400 |
| "Preset11"     | BKN180 BKN320 FEW410 |
| "Preset12"     | BKN120 SCT220 FEW410 |
| "Preset13"     | BKN120 BKN260 FEW410 |
| "Preset14"     | BKN070 FEW410        |
| "Preset15"     | SCT140 BKN240 FEW400 |
| "Preset16"     | BKN140 BKN280 FEW400 |
| "Preset17"     | BKN070 BKN200 BKN320 |
| "Preset18"     | BKN130 BKN250 BKN380 |
| "Preset19"     | OVC090 BKN230 BKN310 |
| "Preset20"     | BKN130 BKN280 FEW380 |
| "Preset21"     | BKN070 OVC170        |
| "Preset22"     | OVC070 BKN170        |
| "Preset23"     | OVC110 BKN180 SCT320 |
| "Preset24"     | OVC030 OVC170 BKN340 |
| "Preset25"     | OVC120 OVC220 OVC400 |
| "Preset26"     | OVC090 BKN230 SCT320 |
| "Preset27"     | OVC080 BKN250 BKN340 |
| "RainyPreset1" | OVC030 OVC280 FEW400 |
| "RainyPreset2" | OVC030 SCT180 FEW400 |
| "RainyPreset3" | OVC060 OVC190 SCT340 |

> [!NOTE]
> The lowest cloud layer's altitude may vary since Real Weather will try to
> match it to the METAR as best as possible.

[1]: https://en.wikipedia.org/wiki/Wind_profile_power_law
[2]: https://www.engineeringtoolbox.com/wind-shear-d_1215.html
[3]: https://en.wikipedia.org/wiki/Wind_gradient#Wind_turbines
