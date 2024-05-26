# Real Weather Bot

## About

Real Weather Bot is a Discord companion bot for DCS Real Weather. The bot
provides an interface to get and set data from the Real Weather utility.

## Setup

To setup the Real Weather Bot, you'll first need to create a new Discord
application through the
[Discord developer hub](https://discord.com/developers/applications). Click
the "New Application" button found at the top right of the portal.

![new application button](/docs/img/new_application.png)

Give the application a name such as "Real Weather Bot". You'll then have to
agree to the developer TOS and developer policy. Click continue once you are
ready.

Once the new application is created, you should be taken to the page to
configure the bot. Here you can add a description to the bot, or click on the
app icon button to add a picture for your Real Weather Bot.

![app icon](/docs/img/app_icon.png)

If you would like to use the official Real Weather icon for your bot, you can
find it in [doc/real_weather.png](/docs/img/dcs_real_weather_icon.png).

Once you have personalized the bot to your liking, click on the "OAuth2"
tab on the left hand side bar. In the OAuth2 URL Generator, select the "bot"
scope, and then add "Send Messages" to the text permissions. It should look like
this:

![oauth](/docs/img/oauth.png)

Then paste the generated URL into your browser and follow the steps to invite
it to your server.

> [!WARNING]
> Before continuing, it is highly recommended that you turn off the "public
> bot" toggle under the bot tab in the developer portal. This will prevent
> others from inviting your instance of Real Weather Bot to their server.

Real Weather Bot should now be in your server! Keep reading to see how to run
and use the bot.

## Usage

Download the latest bot release. Open the botconfig.json in a text editor of
choice.

You'll first need to get your GuildID for the server you invited the bot
to. You can do this enabling Discord developer mode, right clicking on your
server, and then pressing "Copy Server ID." Paste this between the quotes after
`"guild-id":` in the config.

Then get your bot token from the developer portal. Click on the "Bot" page from
the left side, and find the token section. If you've already requested a token
and forgotten it, you'll have to reset it. Otherwise, show the token and save it
into the `"bot-token"` parameter in the config. You will only be able to show
your token once.

Real Weather Bot also needs to know about your Real Weather installation. Copy
the path of your Real Weather executable to the `"real-weather-path"` parameter,
and you log file path to the `"real-weather-log-path"` parameter. Both of these
can take absolute (e.g. `C:/Users/myuser/Desktop/realweather/realweather.exe`)
or relative paths (e.g. `../realweather/realweather.exe`).

> [!NOTE]
> You can also use backslashes `\` in your paths, but if you do you will have
> to escape them with another backslash, so `C:\Users\myuser` would look like
> `C:\\Users\\myuser`. You can also use forward slashes `/` instead without
> having to escape them.

Finally, give your Real Weather Bot a place to put its log file with the `"log"`
parameter. This can also take an absolute or relative path.

Now that everything is configured, you should be able to run the bot and see it
come online in your discord server. If anything was configured wrong, the bot
log should show you what happened.

## Commands

The currently supported bot commands are given below:

- `/set-weather`
  - Allows the user to override weather for the next Real Weather run cycle.
    this command requires you to have the `"use-custom-data"` parameter in your
    Real Weather config set to true. In order for users to run this command,
    they also must have a role called "Real Weather" assigned to them. Any
    weather not overriden with this command will be fetched from CheckWx like
    normal.
- `/last-metar`
  - Fetches and shows the latest METAR from your Real Weather log file.
