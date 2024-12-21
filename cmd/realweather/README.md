# DCS Real Weather

## Usage

1) Optional: To use the CheckWX weather provider:
   - Create an account at [checkwx](https://checkwxapi.com/).
   - Find your API key from your account details and save it for later.
2) [Download the latest Real Weather release][1].
3) Extract the files in the release archive.
4) Open the provided `config.toml` with a text editor of choice (Notepad++ or
Sublime text editor are good choices for starting out).
5) The config file comes with reasonable defaults provided for most options, but
you'll need to configure at least the following settings:
   - `realweather.mission.input`: this is the path to your input mission file to
  edit the weather for.
   - `realweather.mission.output`: this is the path to your output mission that
  will have the new weather.
   - Optional: `api.checkwx.key` and `api.checkwx.enable` if using the CheckWX
  weather provider.
6) Save your changes and ensure the config file remains inside the same
directory as the Real Weather executable.
7) Create or configure the mission file you want to be updated with the real
weather.
8) Launch Real Weather manually by running the executable. If all is configured
correctly, you should see some output in the console as Real Weather goes
through the steps of updating your mission. If any errors are encountered, the
console/log should help you figure out what went wrong, but if extra assistance
is needed, feel free to reach out via [the discussions page][2] or [the
discord][3].
9) Optional: For more advanced configuration, it's recommended to incorporate
Real Weather into your server restart scripts. A simple example is provided in
[examples](/examples/starting_scripts). Even more advanced users may want to
consider something like [DCS Server Bot][4]. DCS Server Bot is a separately
maintained program, please see DCS SB resources for any related troubleshooting
or questions.
10) Enjoy automatic weather updates to your mission!

[1]: https://github.com/evogelsa/DCS-real-weather/releases/latest
[2]: https://github.com/evogelsa/dcs-real-weather/discussions
[3]: https://discord.com/invite/mjr2SpFuqq
[4]: https://github.com/Special-K-s-Flightsim-Bots/DCSServerBot

> [!NOTE]
> Generally your input file should be different from your output file. This will
> leave your main mission template untouched.

## Config file

The config file is the primary way to modify and tweak how Real Weather edits
your mission. The default configuration can be viewed in
[config/config.toml](/config/config.toml). This config contains comments with a
basic explanation of each parameter, but for more details, see the config
parameters section below. The config file uses the TOML specification. TOML is
meant to be obvious enough to not need a guide, but [the full specification can
be found here][5] if you want to learn more.

[5]: https://toml.io/en/v1.0.0

### Config Parameters

* `realweather`: table
  * This is a section for core configuration of Real Weather.
  * `realweather.mission`: table
    * This is a section for defining how Real Weather accesses your mission.
    * `realweather.mission.input`: string
      * This is a path to the mission you want Real Weather to edit. This should
      generally differ from `realweather.mission.output`. While you can use the
      same input and output files, it's not necessary or recommended.
    * `realweather.mission.output`: string
      * This is a path to the mission you want Real Weather to output.
    * `realweather.mission.brief`: table
      * The brief section details options for updating your mission brief.
      * `realweather.mission.brief.add-metar`: boolean
        * If true, Real Weather will add a METAR to your mission brief.
      * `realweather.mission.brief.insert-key`: string
        * This specifies where Real Weather will insert the METAR into your
        brief (if enabled). This key should exist in your mission brief, and the
        following line is where the METAR will be placed. It is important that
        the key is valid when used in a PCRE. You can verify this by typing your
        key into a website like [regex101](https://regex101.com/)
      * `realweather.mission.brief.remarks`: string
        * This will add a remarks section to your METAR. There is no functional
        impact of this, and it is purely for you to customize your METAR with
        extra information if your choose. Feel free to set it to an empty string
        `""` if you don't want it.
  * `realweather.log`: table
    * This is a section for customizing the log behavior of Real Weather.
    * `realweather.log.enable`: boolean
      * Enables logging to a file. Real Weather will always log to the console.
    * `realweather.log.file`: string
      * Path of the file to write the log to
* `api`: table
  * The API section defines how Real Weather will get data to translate into
  your mission.
  * `api.provider-priority`: string array
    * The provider priority array defines how to prioritize different sources of
    METAR data. This array should always contain all the METAR providers
    (currently `"aviationweather"`, `"checkwx"`, and `"custom"`). The first
    source listed will be used first unless it is unreachable or there is an
    error, then the second will be tried, etc. In most scenarios, only the first
    provider listed here will be used.
  * `api.aviationweather`: table
    * This section is used for configuring
    [aviationweather.gov](https://aviationweather.gov/) as a METAR data
    provider.
    * `api.aviationweather.enable`: boolean
      * Enables or disables the aviationweather data provider.
  * `api.checkwx`: table
    * This section configures [checkwx.com](https://www.checkwx.com/) as a METAR
    data provider.
    * `api.checkwx.enable`: boolean
      * Enables or disables checkwx as a data provider.
    * `api.checkwx.key`: string
      * CheckWX requires an API key to access its data. This can be obtained for
        free at [checkwxapi.com](https://www.checkwxapi.com/).
  * `api.custom`: table
    * This defines the custom data provider. The custom provider allows you to
    provide your own weather data to Real Weather through a .json file. An
    example of this weather data can be found in
    [examples/weather_data.json](/examples/weather_data.json). The data in this
    file follows the same format as CheckWX data, so you can also reference
    the [checkwx
    documentation](https://www.checkwxapi.com/documentation/metar#metar-fields)
    for more information. Some important notes regarding custom weather data:
      * Only the fields supplied in the example file are currently supported.
      * Custom weather need not supply every field, though anything omitted and
      not supplied by another data provider will have some default value
      applied. Custom weather is considered an advanced feature, so please ask
      questions if needed via [discussions][2] or [Discord][3].
      * The number of results must be at least 1 in the custom data in order for
      it to be parsed.
    * `api.custom.enable`: boolean
      * Enables or disables the custom weather provider.
    * `api.custom.file`: string
      * Path to the file containing your custom data.
    * `api.custom.override`: boolean
      * This parameter changes how the custom provider is used. If `true`, the
      custom provider will be used as a source of data that overrides anything
      found from the other providers. For example if you only have a barometer
      setting defined in the custom weather data, and CheckWX is your primary
      provider (first in priority list), then all data from CheckWX will be
      used, except the custom barometer setting will override anything CheckWX
      supplies.
  * `api.openmeteo`: table
    * The Open Meteo API is currently the only non-METAR providing API. This API
    is used for getting winds aloft data if enabled. If disabled, Real Weather
    will instead estimate winds aloft using ground wind information (not as
    accurate).
* `options`: table
  * The options section is used for configuring various behaviors of Real
  Weather.
  * `options.time`: table
    * These settings determine how Real Weather updates the mission time
    * `options.time.enable`: boolean
      * This setting enables or disables Real Weather's time updating. If
      enabled, Real Weather will update the time of your mission.
    * `options.time.system-time`: boolean
      * If true, when Real Weather updates mission time, it will use the
      server's system time instead of the METAR report time.
    * `options.time.offset`: string
      * This setting allows you to configure an offset from the system or METAR
      time to use when applying to the mission file. The format of this offset
      is a sequence of numbers, each a unit suffix, such as "-1.5h" or "2h45m".
      Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
  * `options.date`: table
    * These settings determine how Real Weather updates the mission date
    * `options.date.enable`: boolean
      * This setting enables or disables Real Weather's date updating. If
      enabled, Real Weather will update the date of your mission.
    * `options.date.system-date`: boolean
      * If true, when Real Weather updates mission date, it will use the
      server's system date instead of the METAR report date.
    * `options.date.offset`: string
      * This setting allows you to configure an offset from the system or METAR
      date to use when applying to the mission file. The format of this offset
      is a sequence of numbers, each a unit suffix, such as "-1d" or "2m5d".
      Valid date units are "d", "m", and "y" where days are 24 hours, months are
      30 days, and years are 365 days.
  * `options.weather`: table
    * This section defines options for how Real Weather updates the weather.
    * `options.weather.enable`: boolean
      * This enables or disables Real Weather's weather updating. If enabled,
      Real Weather will update the weather of your mission.
    * `options.weather.icao`: string
      * This is the ICAO of the airport you would like weather to be pulled
      from. This option is mutually exclusive with `icao-list`; if both are
      supplied, `icao` will be used. To use `icao-list`, set this to `""`.
    * `options.weather.icao-list`: string array
      * This is a list of ICAOs to randomly choose to fetch weather data from.
        This option is mutually exclusive with `icao`; if both are supplied,
        `icao` will be used. Set `icao` to `""` to use `icao-list`.
    * `options.weather.runway-elevation`: number
      * This is the runway/airport elevation of the ICAO configured in meters.
      This is used to convert cloud heights between MSL and AGL values for METAR
      accuracy. Using this value the METAR will report the clouds in hundreds of
      feet AGL. Additionally the elevation is used to properly represent the QNH
      in DCS. This value will also be used for the reference height in wind
      calculation if the Open Meteo API is disabled and
      `options.weather.wind.fixed-reference` is false.
    * `options.weather.wind`: table
      * This section defines wind specific weather options.
      * `options.weather.wind.enable`: boolean
        * This option enables updating the wind of the mission.
      * `options.weather.wind.minimum`: number
        * This defines the minimum wind speed in meters per second Real Weather
        will set for any altitude. This must be at least 0.
      * `options.weather.wind.maximum`: number
        * This defines the maximum wind speed in meters per second Real Weather
        will set for any altitude. This must be at most 50.
      * `options.weather.wind.gust-minimum`: number
        * This defines the minimum gust speed (also known as ground turbulence
        in the mission editor) in meters per second Real Weather will set. This
        must be at least 0.
      * `options.weather.wind.gust-maximum`: number
        * This defines the maximum gust speed (also known as ground turbulence
        in the mission editor) in meters per second Real Weather will set. This
        must be at most 50.
      * `options.weather.wind.stability`: number
        * This is an advanced configuration option to set the simulated
        atmospheric stability for Real Weather. This value is used when
        estimating wind aloft speeds if the Open Meteo provider is not enabled.
        0.143 is considered neutrally stable, and is generally a reasonable
        default. See the tip below for more information.
      * `options.weather.wind.fixed-reference`: boolean
        * This is an advanced configuration option that changes how Real Weather
        estimates wind aloft speeds when not using Open Meteo for winds aloft.
        If true, Real Weather will use a fixed reference point instead of the
        runway elevation when calculating winds aloft.
    * `options.weather.clouds`: table
      * This section defines cloud specific weather options.
      * `options.weather.clouds.enable`: boolean
        * Enables or disables updating the clouds (and the precipitation) of the
        mission.
      * `options.weather.clouds.fallback-to-legacy`: boolean
        * If this is true, Real Weather will use the legacy weather (no preset)
          when a suitable weather preset is not found.
      * `options.weather.clouds.base`: table
        * This section defines options for cloud bases.
          * `options.weather.clouds.base.minimum`: number
            * This option sets the minimum cloud base in meters above ground
            level that will be set when adding clouds to the mission. This
            option should be at least 0.
          * `options.weather.clouds.base.maximum`: number
            * This option sets the maximum cloud base in meters above ground
            level that will be set when adding clouds to the mission. This
            option should be at most 15000. Currently there are no presets that
            go above 6000.
      * `options.weather.clouds.presets`: table
        * This section defines options for cloud presets.
        * `options.weather.clouds.presets.default`: string
          * This option defines the default cloud preset to use. This is used
          when no preset that matches the METAR cloud conditions is found and
          `fallback-to-legacy` is also false. This can be set to any one of the
          presets in the [preset table](#preset-table), or it can also be set to
          an empty string `""` to default to clear weather.
        * `options.weather.clouds.presets.disallowed`: string array
          * This option defines a list of any presets that you want to be
          prohibited. Any preset in this list will not be a viable option for
          Real Weather to choose. For example, you may want to disable rainy
          presets. Any preset in the [preset table](#preset-table) can be
          contained in this list, or there can be no presets if you want all
          presets to be an option.
    * `options.weather.fog`: table
      * This section defines fog specific weather settings.
      * `options.weather.fog.enable`: boolean
        * This section enables or disables updating the mission fog.
      * `options.weather.fog.mode`: string
        * This may be one of "auto", "manual", or "legacy". Auto and manual
        modes are the new DCS fog options. Legacy is the old fog system which
        seems to be automatically converted by DCS to the equivalent of manual.
        Auto will likely give the best experience while manual may give a closer
        representation to the METAR.
      * `options.weather.fog.thickness-minimum`: number
        * This option defines the minimum fog thickness in meters Real Weather
        will set when setting fog. This must be at least 0. Fog thickness is not
        reported by a METAR, so the thickness set in the mission will be a
        random value between your defined thickness minimum and maximum.
      * `options.weather.fog.thickness-maximum`: number
        * This option defines the maximum fog thickness in meters Real Weather
        will set when setting fog. This must be at most 1000. Fog thickness is
        not reported by a METAR, so the thickness set in the mission will be a
        random value between your defined thickness minimum and maximum.
      * `options.weather.fog.visibility-minimum`: number
        * This option defines the minimum fog visibility in meters that will be
        set when setting fog. This must be at least 0.
      * `options.weather.fog.visibility-maximum`: number
        * This option defines the maximum fog visibility in meters that will be
        set when setting fog. This must be at most 6000.
    * `options.weather.dust`: table
      * This section defines dust specific weather settings.
      * `options.weather.dust.enable`: boolean
        * This option enables or disables updating the mission dust setting.
      * `options.weather.dust.visibility-minimum`: number
        * This option defines the minimum dust visibility in meters that will be
        set when setting dust. This must be at least 300.
      * `options.weather.dust.visibility-maximum`: number
        * This option defines the maximum dust visibility in meters that will be
        set when setting dust. This must be at most 3000.
    * `options.weather.temperature`: table
      * This section defines temperature specific settings.
      * `options.weather.temperature.enable`: boolean
        * This enables or disables setting the mission temperature.
    * `options.weather.pressure`: table
      * This section defines pressure specific settings.
      * `options.weather.pressure.enable`: boolean
        * This enables or disables setting the mission pressure.

> [!IMPORTANT]
> Windows, unlike every other operating system, tends to use backslashes
> `\` in its file paths. If you choose to use backslashes for paths in your
> config file, you must escape them with another backslash, so `C:\Users\myuser`
> would become `C:\\Users\\myuser`. Alternatively, you can use a forward slash
> without escaping it, e.g. `C:/Users/myuser`.

> [!TIP]
> For more info on stability, consider referencing this Wikipedia article about
> [the wind profile power law][6], [this page on wind shear][7], or the wind
> turbines section of the Wikipedia article on [wind gradient][8].

[6]: https://en.wikipedia.org/wiki/Wind_profile_power_law
[7]: https://www.engineeringtoolbox.com/wind-shear-d_1215.html
[8]: https://en.wikipedia.org/wiki/Wind_gradient#Wind_turbines

### Preset table

The supported DCS presets are shown in following table. These preset names are
the same names that can be added to the disallowed presets and default preset
config parameters.

| Preset Name      | Cloud Layers         | Precipitation   |
|------------------|----------------------|-----------------|
| "Preset1"        | FEW070               | None            |
| "Preset2"        | FEW080 SCT230        | None            |
| "Preset3"        | SCT080 FEW210        | None            |
| "Preset4"        | SCT080 SCT240        | None            |
| "Preset5"        | SCT140 FEW270 BKN400 | None            |
| "Preset6"        | SCT080 FEW400        | None            |
| "Preset7"        | BKN075 SCT210 SCT400 | None            |
| "Preset8"        | SCT180 FEW360 FEW400 | None            |
| "Preset9"        | BKN075 SCT200 FEW410 | None            |
| "Preset10"       | SCT180 FEW360 FEW400 | None            |
| "Preset11"       | BKN180 BKN320 FEW410 | None            |
| "Preset12"       | BKN120 SCT220 FEW410 | None            |
| "Preset13"       | BKN120 BKN260 FEW410 | None            |
| "Preset14"       | BKN070 FEW410        | None            |
| "Preset15"       | SCT140 BKN240 FEW400 | None            |
| "Preset16"       | BKN140 BKN280 FEW400 | None            |
| "Preset17"       | BKN070 BKN200 BKN320 | None            |
| "Preset18"       | BKN130 BKN250 BKN380 | None            |
| "Preset19"       | OVC090 BKN230 BKN310 | None            |
| "Preset20"       | BKN130 BKN280 FEW380 | None            |
| "Preset21"       | BKN070 OVC170        | None            |
| "Preset22"       | OVC070 BKN170        | None            |
| "Preset23"       | OVC110 BKN180 SCT320 | None            |
| "Preset24"       | OVC030 OVC170 BKN340 | None            |
| "Preset25"       | OVC120 OVC220 OVC400 | None            |
| "Preset26"       | OVC090 BKN230 SCT320 | None            |
| "Preset27"       | OVC080 BKN250 BKN340 | None            |
| "RainyPreset1"   | OVC030 OVC280 FEW400 | Rain/snow       |
| "RainyPreset2"   | OVC030 SCT180 FEW400 | Rain/snow       |
| "RainyPreset3"   | OVC060 OVC190 SCT340 | Rain/snow       |
| "RainyPreset4"   | SCT080 FEW360        | Light rain/snow |
| "RainyPreset5"   | BKN070 BKN200 BKN320 | Light rain/snow |
| "RainyPreset6"   | OVC090 BKN230 BKN310 | Light rain/snow |
| "NEWRAINPRESET4" | SCT080 SCT120        | Rain/snow       |

> [!NOTE]
> The lowest cloud layer's altitude will vary since Real Weather will try to
> match it to the METAR as best as possible.

## Command line interface

There are a few options that can be passed to Real Weather via a command line
interface. Generally these are for advanced use, and they are not necessary in
order to use Real Weather. Command line options can be passed with either a
single `-` or two `--`. Boolean options can be set to true by simply passing the
flag. For example, `-help` is equivalent to `-help=true`. Other types can be set
with an equals or with a space, for example `-config="myconfig.toml"` is the
same as `-config myconfig.toml`.

### CLI Options

```
Usage of realweather:
	Boolean Flags:
		-enable-custom  forcibly enable the custom weather provider
		-help           prints this help message
		-validate       validates your config the exits
		-version        prints the Real Weather version then exits

	String Flags:
		-config         override default config file name
		-custom-file    override file path for custom weather provider
		-icao           override icao
		-input          override input mission
		-output         override output mission
```
