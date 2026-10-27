$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot

function New-RandomSecret {
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    return ([System.BitConverter]::ToString($bytes)).Replace('-', '').ToLowerInvariant()
}

function Set-EnvValue([string]$path, [string]$key, [string]$value) {
    $text = [System.IO.File]::ReadAllText($path)
    $pattern = '(?m)^' + [regex]::Escape($key) + '=.*$'
    if ([regex]::IsMatch($text, $pattern)) {
        $text = [regex]::Replace($text, $pattern, ($key + '=' + $value))
    } else {
        $text += "`n$key=$value`n"
    }
    [System.IO.File]::WriteAllText($path, $text, (New-Object System.Text.UTF8Encoding($false)))
}

$created = @{}
foreach ($relative in @('platform/backend', 'platform/frontend', 'platform/ai', 'knowledge')) {
    $directory = Join-Path $repoRoot $relative
    $target = Join-Path $directory '.env'
    if (-not (Test-Path -LiteralPath $target)) {
        Copy-Item -LiteralPath (Join-Path $directory '.env.example') -Destination $target
        $created[$relative] = $true
        Write-Host "Created $relative/.env"
    } else {
        Write-Host "Preserved $relative/.env"
    }
}

$backend = Join-Path $repoRoot 'platform/backend/.env'
$knowledge = Join-Path $repoRoot 'knowledge/.env'
if ($created.ContainsKey('platform/backend')) {
    Set-EnvValue $backend 'JWT_ACCESS_SECRET' (New-RandomSecret)
    Set-EnvValue $backend 'JWT_REFRESH_SECRET' (New-RandomSecret)
    Set-EnvValue $backend 'MICROSOFT_TOKEN_ENCRYPTION_KEY' (New-RandomSecret)
}

if ($created.ContainsKey('platform/backend') -and $created.ContainsKey('knowledge')) {
    $sharedKey = New-RandomSecret
    Set-EnvValue $backend 'KNOWLEDGE_SERVICE_API_KEY' $sharedKey
    Set-EnvValue $knowledge 'KNOWLEDGE_INTERNAL_API_KEY' $sharedKey
} elseif ($created.ContainsKey('platform/backend') -or $created.ContainsKey('knowledge')) {
    Write-Warning 'Only one service config was newly created. Align the knowledge API keys manually; existing config was preserved.'
}

Write-Host 'Local configuration is ready. Fill model keys and Microsoft settings before use. No services were started.'
