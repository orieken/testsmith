Write-Host "══════════════════════════════════════"
Write-Host "  TestSmith — Local Binary Build"
Write-Host "══════════════════════════════════════"

poetry install --with build

Remove-Item -Recurse -Force build/, dist/ -ErrorAction SilentlyContinue

Write-Host "`nBuilding binary..."
poetry run pyinstaller testsmith.spec --clean --noconfirm

Write-Host "`nVerifying..."
& .\dist\testsmith.exe --version

$size = (Get-Item .\dist\testsmith.exe).Length / 1MB
Write-Host "`n══════════════════════════════════════"
Write-Host "  Build complete!"
Write-Host "  Binary: dist\testsmith.exe"
Write-Host ("  Size:   {0:N1} MB" -f $size)
Write-Host "══════════════════════════════════════"
