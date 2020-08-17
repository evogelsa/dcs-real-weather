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

### Config file parameters

| Key         | Type   | Description                                        |
|-------------|--------|----------------------------------------------------|
| api-key     | string | API key from [checkwx](https://www.checkwxapi.com) |
| icao        | string | airport ICAO where you want to get METAR from      |
| hour-offset | int    | mission time offset from system time               |
| input-file  | string | mission file which will be used to modify          |
| output-file | string | name of mission file which will be output          |

After you have edited the config to your liking it is recommended you place the
binary into its own directory along with the config file. You can either
manually move the `input-file` into this directory and run the binary, or you
can accomplish this task through a script. An example batch script is provided
in [examples](./examples/replace_mission.bat)
