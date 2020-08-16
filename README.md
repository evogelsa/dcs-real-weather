# DCS Real Weather Updater

## About

Replaces static weather data in DSC mission file with METAR reported data from
ICAO of choice.

Not extensively tested.

## Usage

Edit config.json and replace API key with one generated from
[checkwx](https://www.checkwxapi.com/). Update ICAO with icao of airport of your
choosing. By default the program will set the mission time to system time, but
this can be adjusted by changing the time-offset in config file to an integer.
This integer will ofset time by x hours.

Put binary into own directory with config and move "mission.miz" into directory
before executing. It will produce a new missionfile called mission.miz.

It is recommended to use an external script to call this program. An example
batch file is provided inside /examples. Note that the binary will expect the
curent working directory to be inside its parent directory.
