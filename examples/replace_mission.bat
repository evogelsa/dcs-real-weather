@echo off

:: Assumes following directory structure:
::  - parent-dir/
::      - Missions/
::          - backups/
::          - mission.miz
::      - realweather/
::          - realweather.exe
::      - replace_mission.bat

:: Make backup of current mission file
copy "Missions\mission.miz" "Missions\backups\mission.miz"
set d=%date:~-4,4%%date:~0,2%%date:~-7,2%
set d=%d: =_%
set t=%time:~0,2%%time:~3,2%%time:~6,2%
set t=%t: =0%
rename "Missions\backups\mission.miz" "%d%_%t%_mission.miz"

:: Move mission into realweather working directory and move updated mission
:: back into missions folder
move "Missions\mission.miz" "realweather\mission.miz" :: realweather expects "mission.miz"
cd realweather                                        :: CWD must be parent directory of binary
call realweather.exe
cd ..
move "realweather\realweather.miz" "Missions\DSMC_Gorgas_001.miz"
