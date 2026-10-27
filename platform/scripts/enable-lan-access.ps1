$ErrorActionPreference = "Stop"

$ports = @(3000, 8080, 9100)
$rulePrefix = "FinTechBase-LAN"

Write-Host "Configuring Windows Firewall rules for FinTechBase LAN access..." -ForegroundColor Cyan

foreach ($port in $ports) {
  $name = "$rulePrefix-$port"
  $existing = Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue
  if (-not $existing) {
    New-NetFirewallRule `
      -DisplayName $name `
      -Direction Inbound `
      -Action Allow `
      -Protocol TCP `
      -Profile Private `
      -LocalPort $port | Out-Null
    Write-Host "Added firewall rule: $name" -ForegroundColor Green
  } else {
    Write-Host "Firewall rule already exists: $name" -ForegroundColor Yellow
  }
}

$ipv4 = Get-NetIPAddress -AddressFamily IPv4 |
  Where-Object {
    $_.IPAddress -notlike "127.*" -and
    $_.IPAddress -notlike "169.254.*" -and
    $_.PrefixOrigin -ne "WellKnown"
  } |
  Select-Object -First 1 -ExpandProperty IPAddress

Write-Host ""
Write-Host "LAN access is ready." -ForegroundColor Green
if ($ipv4) {
  Write-Host "Frontend URL: http://$ipv4`:3000"
  Write-Host "Backend URL:  http://$ipv4`:8080"
  Write-Host "AI URL:       http://$ipv4`:9100"
} else {
  Write-Host "Could not detect a usable LAN IPv4 address automatically." -ForegroundColor Yellow
}
