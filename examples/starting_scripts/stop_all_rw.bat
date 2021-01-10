@echo off

taskkill /f /im DCS.exe
taskkill /f /im SR-Server.exe
taskkill /f /im Perun.exe
taskkill /f /t /fi "WINDOWTITLE eq DCS-TASK-LOOP"