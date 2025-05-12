@echo off
echo Building xmlui-launcher.exe...
go build -o xmlui-launcher.exe xmlui-launcher.go
if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)
echo Build succeeded: xmlui-launcher.exe
