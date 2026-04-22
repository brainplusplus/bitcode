@echo off
cd /d "%~dp0..\..\engine"
go build -o bin\engine.exe cmd\engine\main.go
