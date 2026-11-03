# Crea el rol y la base de datos de PostgreSQL para este template y
# actualiza back/.env con las credenciales de la aplicación.
#
# Uso:
#   powershell -ExecutionPolicy Bypass -File .\scripts\setup-postgres.ps1
#
# El script pide la contraseña del superusuario de PostgreSQL (por defecto
# "postgres") de forma segura. No guarda esa contraseña: solo la usa para
# crear el rol de la aplicación, la base de datos y los permisos.
#
# Valores por defecto:
#   Base de datos : mydbd
#   Usuario app   : template_app
#   Puerto        : 5432
#   Host          : localhost
#
# Puedes cambiarlos con parámetros, por ejemplo:
#   .\scripts\setup-postgres.ps1 -DbName template -AppUser angular_go -Port 5433

param(
    [string]$PgBin = "C:\Program Files\PostgreSQL\18\bin",
    [string]$DbHost = "localhost",
    [int]$Port = 5432,
    [string]$SuperUser = "postgres",
    [string]$DbName = "mydbd",
    [string]$AppUser = "template_app"
)

$ErrorActionPreference = "Stop"

$psql = Join-Path $PgBin "psql.exe"
if (-not (Test-Path $psql)) {
    throw "No se encontro psql.exe en '$PgBin'. Ajusta -PgBin con la ruta de tu instalacion."
}

function Invoke-Psql {
    param(
        [string]$Database,
        [string]$Sql,
        [switch]$Quiet
    )

    $psqlArgs = @(
        "-h", $DbHost,
        "-p", "$Port",
        "-U", $SuperUser,
        "-d", $Database,
        "-v", "ON_ERROR_STOP=1"
    )
    if ($Quiet) { $psqlArgs += "-tA" }
    $psqlArgs += @("-c", $Sql)

    & $psql @psqlArgs
    if ($LASTEXITCODE -ne 0) {
        throw "psql fallo (codigo $LASTEXITCODE) ejecutando: $Sql"
    }
}

function New-AppPassword {
    $alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
    $bytes = New-Object byte[] 28
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $rng.GetBytes($bytes)
    } finally {
        $rng.Dispose()
    }
    return -join ($bytes | ForEach-Object { $alphabet[$_ % $alphabet.Length] })
}

function Set-EnvValue {
    param(
        [string]$Content,
        [string]$Name,
        [string]$Value
    )

    $pattern = "(?m)^" + [regex]::Escape($Name) + "=.*$"
    $line = "$Name=$Value"
    if ($Content -match $pattern) {
        return [regex]::Replace($Content, $pattern, $line)
    }
    return $Content.TrimEnd() + "`r`n" + $line + "`r`n"
}

Write-Host ""
Write-Host "Configuracion de PostgreSQL para el template" -ForegroundColor Cyan
Write-Host "  Host      : $DbHost"
Write-Host "  Puerto    : $Port"
Write-Host "  Base      : $DbName"
Write-Host "  Usuario   : $AppUser"
Write-Host ""

$securePassword = Read-Host "Password del superusuario '$SuperUser'" -AsSecureString
$bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
try {
    $superPassword = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
} finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
}

if ([string]::IsNullOrWhiteSpace($superPassword)) {
    throw "La password del superusuario no puede estar vacia."
}

$env:PGPASSWORD = $superPassword

try {
    Invoke-Psql -Database "postgres" -Sql "SELECT 1" -Quiet | Out-Null
    Write-Host "Conexion con '$SuperUser' verificada." -ForegroundColor Green

    $appPassword = New-AppPassword

    $roleExists = (Invoke-Psql -Database "postgres" -Quiet -Sql "SELECT 1 FROM pg_roles WHERE rolname = '$AppUser'").Trim()
    if ($roleExists -eq "1") {
        Invoke-Psql -Database "postgres" -Sql "ALTER ROLE $AppUser WITH LOGIN PASSWORD '$appPassword';" | Out-Null
        Write-Host "Rol '$AppUser' actualizado." -ForegroundColor Green
    } else {
        Invoke-Psql -Database "postgres" -Sql "CREATE ROLE $AppUser LOGIN PASSWORD '$appPassword';" | Out-Null
        Write-Host "Rol '$AppUser' creado." -ForegroundColor Green
    }

    $dbExists = (Invoke-Psql -Database "postgres" -Quiet -Sql "SELECT 1 FROM pg_database WHERE datname = '$DbName'").Trim()
    if ($dbExists -ne "1") {
        Invoke-Psql -Database "postgres" -Sql "CREATE DATABASE $DbName OWNER $AppUser;" | Out-Null
        Write-Host "Base de datos '$DbName' creada." -ForegroundColor Green
    } else {
        Write-Host "La base de datos '$DbName' ya existe; se conservan sus datos." -ForegroundColor Yellow
        Invoke-Psql -Database $DbName -Sql "GRANT CONNECT ON DATABASE $DbName TO $AppUser;" | Out-Null
        Invoke-Psql -Database $DbName -Sql "GRANT USAGE, CREATE ON SCHEMA public TO $AppUser;" | Out-Null
        Invoke-Psql -Database $DbName -Sql "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $AppUser;" | Out-Null
        Invoke-Psql -Database $DbName -Sql "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $AppUser;" | Out-Null
        Invoke-Psql -Database $DbName -Sql "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $AppUser;" | Out-Null
        Invoke-Psql -Database $DbName -Sql "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $AppUser;" | Out-Null
        Write-Host "Permisos otorgados a '$AppUser' sobre '$DbName'." -ForegroundColor Green
    }

    # Verificacion con las credenciales de la aplicacion.
    $env:PGPASSWORD = $appPassword
    $check = (Invoke-Psql -Database $DbName -Quiet -Sql "SELECT current_user || ' @ ' || current_database();").Trim()
    if ($check -ne "$AppUser @ $DbName") {
        throw "La verificacion con el usuario de la aplicacion devolvio: $check"
    }
    Write-Host "Verificacion de la app: $check" -ForegroundColor Green

    # Actualizar back/.env sin tocar el resto de variables.
    $projectRoot = Split-Path $PSScriptRoot -Parent
    $envPath = Join-Path $projectRoot ".env"
    if (-not (Test-Path $envPath)) {
        throw "No se encontro '$envPath'."
    }

    $content = Get-Content -Path $envPath -Raw
    $content = [regex]::Replace($content, "(?m)^USE_MOCK=.*\r?\n?", "")
    $content = Set-EnvValue -Content $content -Name "DB_TYPE" -Value "postgres"
    $content = Set-EnvValue -Content $content -Name "DB_HOST" -Value $DbHost
    $content = Set-EnvValue -Content $content -Name "DB_PORT" -Value "$Port"
    $content = Set-EnvValue -Content $content -Name "DB_USER" -Value $AppUser
    $content = Set-EnvValue -Content $content -Name "DB_PASSWORD" -Value $appPassword
    $content = Set-EnvValue -Content $content -Name "DB_NAME" -Value $DbName
    $content = Set-EnvValue -Content $content -Name "DB_SSLMODE" -Value "disable"

    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($envPath, $content, $utf8NoBom)

    Write-Host ""
    Write-Host "Listo. Credenciales escritas en: $envPath" -ForegroundColor Green
    Write-Host "  DB_USER=$AppUser"
    Write-Host "  DB_NAME=$DbName"
    Write-Host "  DB_PASSWORD=(guardada en .env)"
    Write-Host ""
    Write-Host "Siguiente paso: cd back; air (o go run ./cmd/server)" -ForegroundColor Cyan
} finally {
    Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
}
