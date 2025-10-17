@echo off
REM Script de build pour Windows
setlocal enabledelayedexpansion

REM Configuration
set PROJECT_NAME=remoteshell
set VERSION=%VERSION%
if "%VERSION%"=="" set VERSION=1.0.0
set BUILD_DIR=build
set DIST_DIR=dist

echo [INFO] Build RemoteShell v%VERSION%

REM Vérifier que Go est installé
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go n'est pas installé. Veuillez installer Go 1.21 ou plus récent.
    exit /b 1
)

REM Vérifier que Node.js est installé
node --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Node.js n'est pas installé. Veuillez installer Node.js 18 ou plus récent.
    exit /b 1
)

REM Nettoyer les anciens builds
echo [INFO] Nettoyage des anciens builds...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
if exist %DIST_DIR% rmdir /s /q %DIST_DIR%
mkdir %BUILD_DIR%
mkdir %DIST_DIR%

REM Build de l'interface web
echo [INFO] Build de l'interface web...
cd web
if not exist node_modules (
    echo [INFO] Installation des dépendances npm...
    npm install
)
echo [INFO] Build de production React...
npm run build
xcopy /E /I dist ..\%BUILD_DIR%\web\
cd ..
echo [SUCCESS] Interface web buildée avec succès

REM Build des binaires Go
echo [INFO] Build des binaires Go...

REM Windows AMD64
echo [INFO] Build pour windows/amd64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\server-windows-amd64.exe ./cmd/server
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\agent-windows-amd64.exe ./cmd/agent

REM Windows ARM64
echo [INFO] Build pour windows/arm64...
set GOOS=windows
set GOARCH=arm64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\server-windows-arm64.exe ./cmd/server
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\agent-windows-arm64.exe ./cmd/agent

REM Linux AMD64
echo [INFO] Build pour linux/amd64...
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\server-linux-amd64 ./cmd/server
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\agent-linux-amd64 ./cmd/agent

REM Linux ARM64
echo [INFO] Build pour linux/arm64...
set GOOS=linux
set GOARCH=arm64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\server-linux-arm64 ./cmd/server
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\agent-linux-arm64 ./cmd/agent

REM Darwin AMD64
echo [INFO] Build pour darwin/amd64...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\server-darwin-amd64 ./cmd/server
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\agent-darwin-amd64 ./cmd/agent

REM Darwin ARM64
echo [INFO] Build pour darwin/arm64...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\server-darwin-arm64 ./cmd/server
go build -ldflags "-X main.version=%VERSION%" -o %BUILD_DIR%\agent-darwin-arm64 ./cmd/agent

echo [SUCCESS] Build terminé avec succès!
echo [INFO] Les binaires sont disponibles dans le répertoire %BUILD_DIR%


