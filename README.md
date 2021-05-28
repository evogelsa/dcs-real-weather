# DCS Real Weather Updater

## About

Replaces static weather data in a DCS mission file with data from a METAR report
of an airport of choice.

Not extensively tested.

## Usage

First step is to create an API key at [checkwx](https://www.checkwxapi.com/).
This will be needed to fetch the weather data. You'll then want to edit the
`config.json` file provided in the zip download. Each field is required and a
summary of their purpose is provided below. Some reasonable defaults are
provided for most options.

Second step is to create the mission file you want to be updated with the METAR
weather. The input mission file you supply must have one of the preset cloud
types selected in order for it to update properly, but the other weather options
should not matter.

### Config file parameters

| Key                 | Type   | Description                                   |
|---------------------|--------|-----------------------------------------------|
| api-key             | string | API key [checkwx](https://www.checkwxapi.com) |
| icao                | string | airport ICAO where you want to get METAR from |
| hour-offset         | int    | mission time offset from system time          |
| input-mission-file  | string | mission file which will be used to modify     |
| output-mission-file | string | name of mission file which will be output     |
| update-time         | bool   | whether or not to update time of mission      |
| update-weather      | bool   | whether or not to update weather of mission   |
| logfile             | string | name log, "" will disable logfile             |
| metar-remarks       | string | remarks to add to metar                       |

After you have edited the config to your liking it is recommended you place the
binary into its own directory along with the config file. You can either
manually move the `input-file` into this directory and run the binary, or you
can accomplish this task through a script. An example batch script is provided
in [examples](./examples)
