# DCS Real Weather Updater

## About

Replaces static weather data in a DCS mission file with data from a METAR report
of an airport of choice.

Not extensively tested.

## Usage

1) Create an account at [checkwx](https://checkwxapi.com/).
2) Find your API key from your account details and copy this.
3) Download the [latest release](/releases/latest/).
4) Edit the `config.json` provided in the zip. Add your API key and configure
the other settings to your liking. A description of each of the settings is
provided [below](#config-file-parameters).
5) Place the config file inside the same directory that the realweather.exe file
is located.
6) Create or configure the mission file you want to be updated with the real
weather. **Important: the mission file you want to be updated must have a cloud
preset already selected in order to work. This cloud preset will be changed
after running the utility.**
7) To run the utility you may either manually run the realweather.exe, or you
can use a script to automate the process. Some examples are provided in
[examples](/examples).

## Notes

* Remember to have a cloud preset already selected in the input mission file.
This preset will get changed when the realweather utility updates the mission,
but having the preset selected is important for ensuring the structure of the
mission file is correct.
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
| input-mission-file  | string | path of the mission file to be updated        |
| output-mission-file | string | path of the mission file that will be output  |
| update-time         | bool   | whether or not to update time of mission      |
| update-weather      | bool   | whether or not to update weather of mission   |
| logfile             | string | name of log, "" will disable logfile          |
| metar-remarks       | string | remarks to add to metar                       |
