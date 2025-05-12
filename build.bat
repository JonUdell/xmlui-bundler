@echo off
echo Building xmlui-bundler.exe...
go build -o xmlui-bundler.exe xmlui-bundler.go
if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)
echo Build succeeded: xmlui-bundler.exe
