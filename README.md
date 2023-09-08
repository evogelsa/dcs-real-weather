# DCS Real Weather Updater

[![Downloads](https://img.shields.io/github/downloads/evogelsa/DCS-real-weather/total?logo=GitHub)](https://github.com/evogelsa/DCS-real-weather/releases/latest)
[![Latest Release](https://img.shields.io/github/v/release/evogelsa/DCS-real-weather?logo=GitHub)](https://github.com/evogelsa/DCS-real-weather/releases/latest)
[![Discord](https://img.shields.io/discord/1148739727990722751?logo=Discord)](https://discord.com/invite/mjr2SpFuqq)

## About

This program is a simple utility meant to be incorporated into a DCS server's
restart script. The utility fetches the most recent weather report (METAR) from
a selected airport and attempts to make the weather conditions inside a provided
mission file match the report. When configured this way, a server can run a
static mission file but regularly update the weather conditions automatically.
The utility can also update time of day based off the current time and a given
offset if desired.

## Usage

1) Create an account at [checkwx](https://checkwxapi.com/).
2) Find your API key from your account details and copy it.
3) Download the
[latest release](https://github.com/evogelsa/DCS-real-weather/releases/latest).
4) Extract the files in the release zip.
5) Open the provided `config.json` with a text editor of choice.
6) Add your API key between the quotes and configure the other settings to your
liking. A description of each of the settings is provided
[below](#config-file-parameters).
7) Save your changes and ensure the config file remains inside the same
directory that the realweather.exe file is located.
8) Create or configure the mission file you want to be updated with the real
weather.
9) To run the utility you may either manually run the realweather.exe, or you
can use a script to automate the process. Some examples are provided in
[examples](/examples).
10) Enjoy automatic weather updates to your mission!

## Notes

* It is recommended that you keep a master copy of your input mission file, and
    then use this mission for each weather update rather than updating the
    realweather mission. This way your main mission stays safe in the rare
    event of some malfunction leading to corruption. You can accomplish this
    through your normal restarting script, but an example is provided in
    [examples](/examples) if you do not already have something like this set
    up.
* If your input mission file is not in the same directory as realweather.exe,
    make sure that you have an absolute path to the file in your config.
    However, it's recommended that you use a relative path and copy the input
    mission into the realweather directory as part of your server start/restart
    script.

## Config file parameters

The config file looks like the following:

```json
{
  "api-key": "", // your api key from checkwx
  "files": {
    "input-mission": "mission.miz",      // path of mission file to be updated
    "output-mission": "realweather.miz", // path of output mission file
    "log": "logfile.log"                 // path of log file, "" disables
  },
  "metar": {
    "icao": "KDLH", // ICAO of the aiport to fetch METAR from
    "remarks": ""   // addtional remarks to add to METAR, for logging only
  },
  "options": {
    "update-time": true,     // set to false to disable time being updated
    "update-weather": true,  // set to false to disable weather being updated
    "time-offset": "-5h30m", // time offset from system time
    "wind": {
      "minimum": -1,     // maximum allowed wind speed in m/s, negative disables
      "maximum": -1,     // minimum allowed wind speed in m/s, negative disables
      "stability": 0.143 // atmospheric stability used in wind profile power law
    },
    "clouds": {
      "disallowed-presets": [
          "RainyPreset1",
          "RainyPreset2",
          "RainyPreset3"
          ] // List of weather presets you do not want to be chosen
    },
    "fog-allowed": true, // set to false to disable fog
    "dust-allowed": true // set to false to disable dust
  }
}
```

Additional notes:

* For more info on stability, see the following [1][1], [2][2], [3][3].
* Min and max wind speeds can be disabled by setting to a negative number
* Time offset takes a string that can consist of hours, minutes, and seconds.
These are specified by "h", "m", and "s".
* For a list of preset names, see the CloudPresets defined near the bottom of
[`weather.go`](weather/weather.go)

## Contributing

Interested in helping to improve this project? Please see the [contributing
guide](CONTRIBUTING.md) for guidelines on making suggestions, opening new
issues, or contributing code. Thanks for your interest!

## Enjoying this project?

Please consider starring the project to show your support. If you
would like to get more involved, read the [contributing guide](CONTRIBUTING.md)
to see how you can improve DCS Real Weather. This started as a small personal
project and has grown to a small user base over the past couple years. Feel
free to spread the love by posting about DCS real weather or by sharing with
friends. Also join our small [Discord](https://discord.com/invite/mjr2SpFuqq)
community for support, announcements, and camaraderie. For those interested in
supporting the project financially, please see the "sponsor" button at the top
of the page for options. Thanks!

[1]: https://en.wikipedia.org/wiki/Wind_profile_power_law
[2]: https://www.engineeringtoolbox.com/wind-shear-d_1215.html
[3]: https://en.wikipedia.org/wiki/Wind_gradient#Wind_turbines
