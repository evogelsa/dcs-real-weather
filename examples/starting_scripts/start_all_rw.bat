@echo off

rem This script is capable of launching multiple applications including a DCS
rem server after updating a mission with live weather. It then restarts the DCS
rem server every four hours. Start paths are defined below as well as an input
rem and output mission file.

rem DEFINE PATHS BELOW

rem Required: DCS paths
set dcs=C:\Program Files\Eagle Dynamics\DCS World OpenBeta Server\bin\DCS_updater.exe
set inputFile=C:\Users\admin\Saved Games\DCS.openbeta_server\Missions\SVR2_PG_010_AD.miz
set outputFile=C:\Users\admin\Saved Games\DCS.openbeta_server\Missions\SVR2_PG_010_RW.miz

rem Other programs to start

rem Perun
set perun=C:\Users\admin\Desktop\Perun\Perun.application
set lotatcStats=C:\Users\admin\Saved Games\DCS.openbeta_server\Mods\services\LotAtc\stats.json
set srsClients=C:\Program Files\DCS-SimpleRadio-Standalone\clients-list.json

rem SRS
set srsParentDir=C:\Program Files\DCS-SimpleRadio-Standalone



rem Pass 0 to this file when starting to start only DCS, default is start all
if not [%1]==[] (
    set /A startAll=%1
) else (
    set /A startAll=1
)

rem Title command window for easy shutting down via script
title DCS-TASK-LOOP

rem save current working directory as variable
set origDir=%cd%

rem Start Perun and SRS - these do not need to be included in the restart loop
if not %startAll%==0 (
    if not "%perun%"=="" (start "" "%perun%" 48621 1 "%srsClients%" "%lotatcStats%" 1)
    if not "%srsParentDir%"=="" (
        cd "%srsParentDir%"
        start "" "SR-Server.exe"
        cd %origDir%
    )
)

:update
echo **************************************
echo * UPDATING MISSION FILE WITH WEATHER *
echo **************************************

rem Copy desired mission file to working directory
copy "%inputFile%" "%origDir%\realweather\mission.miz"

rem Change CWD to realweather dir and call realweather to update mission
cd "%origDir%\realweather"
call realweather.exe
cd "%origDir%"

rem Move new file back into missions
move "%origDir%\realweather\realweather.miz" "%outputFile%"

rem Clean working directory
del "%origDir%\realweather\mission.miz"

echo Mission updated.

rem Start server
echo **************************************
echo *        STARTING DCS SERVER         *
echo **************************************
start "" /high "%dcs%"
echo Server started.

rem Wait 4 hours for restart
echo **************************************
echo *        WAITING FOR RESTART         *
echo **************************************
timeout /t 14400

rem Stop server
echo **************************************
echo *        STOPPING DCS SERVER         *
echo **************************************
taskkill /f /im DCS.exe
echo Server stopped.
timeout /t 15

rem Refresh weather and repeat
goto update
