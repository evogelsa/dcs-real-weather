# DCS Real Weather Updater

## About

Replaces static weather data in a DCS mission file with data from a METAR report
of an airport of choice.

Not extensively tested.

## Usage

1) Create an account at [checkwx](https://checkwxapi.com/).
2) Find your API key from your account details and copy this.
3) Download the [latest release](https://github.com/evogelsa/DCS-real-weather/releases/latest).
4) Edit the `config.json` provided in the zip. Add your API key and configure
the other settings to your liking. A description of each of the settings is
provided [below](#config-file-parameters).
5) Place the config file inside the same directory that the realweather.exe file
is located.
6) Create or configure the mission file you want to be updated with the real
weather.
7) To run the utility you may either manually run the realweather.exe, or you
can use a script to automate the process. Some examples are provided in
[examples](/examples).

## Notes

* It is recommended that you keep a master copy of your input mission file, and
then reupdate this mission every server restart cycle. You can accomplish this
through your normal restarting script, but an example is provided in
[examples](/examples) if you do not already have something like this set up.
* If your input mission file is not in the same directory as realweather.exe,
make sure that you have an absolute path to the file in your config. However,
it's recommended that you use a relative path and copy the input mission into
the realweather directory as part of your server start/restart script.

## Config file parameters

| Key                 | Type   | Description                                   |
|---------------------|--------|-----------------------------------------------|
| api-key             | string | [checkwx](https://www.checkwxapi.com) API key |
| icao                | string | airport ICAO where you want to get METAR from |
| hour-offset         | int    | mission time offset from system time          |
| stability           | float  | [atmospheric stability number][1]             |
| input-mission-file  | string | path of the mission file to be updated        |
| output-mission-file | string | path of the mission file that will be output  |
| update-time         | bool   | whether or not to update time of mission      |
| update-weather      | bool   | whether or not to update weather of mission   |
| logfile             | string | name of log, "" will disable logfile          |
| metar-remarks       | string | remarks to add to metar                       |

[1]: https://en.wikipedia.org/wiki/Wind_profile_power_law
