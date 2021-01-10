@echo off

taskkill /f /t /fi "WINDOWTITLE eq DCS-TASK-LOOP"
taskkill /f /im SR-Server.exe
taskkill /f /im Perun.exe
taskkill /f /im DCS.exe

timeout /t 10

start start_all_rw.bat 1
