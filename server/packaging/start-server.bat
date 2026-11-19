@echo off
cd /d "%~dp0"
vxaos-server.exe -config config.toml
pause
