param([string]$OutputDirectory = '')

$ErrorActionPreference = 'Stop'
# 只打包运行资源，源码、测试与宿主实现分别交付。
if (-not $OutputDirectory) {
    $OutputDirectory = Join-Path $PSScriptRoot '../../output/extensions'
}
$OutputDirectory = [System.IO.Path]::GetFullPath($OutputDirectory)
& node (Join-Path $PSScriptRoot 'public/source/build.mjs')
if ($LASTEXITCODE -ne 0) { throw '原生页面构建失败' }
$manifest = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'manifest.json') -Raw | ConvertFrom-Json
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
$archivePath = Join-Path $OutputDirectory ($manifest.id + '-' + $manifest.version + '.zip')
Add-Type -AssemblyName System.IO.Compression
$stream = [System.IO.File]::Open($archivePath, [System.IO.FileMode]::Create)
try {
    $archive = New-Object System.IO.Compression.ZipArchive($stream, [System.IO.Compression.ZipArchiveMode]::Create, $true)
    try {
        $files = @('manifest.json', 'README.md', 'public/compat.html')
        $files += Get-ChildItem -LiteralPath (Join-Path $PSScriptRoot 'public/native') -File | Sort-Object Name | ForEach-Object { 'public/native/' + $_.Name }
        foreach ($relative in $files) {
            $entry = $archive.CreateEntry($relative, [System.IO.Compression.CompressionLevel]::Optimal)
            $entry.LastWriteTime = [DateTimeOffset]::new(2026, 9, 18, 0, 0, 0, [TimeSpan]::Zero)
            $entryStream = $entry.Open()
            try {
                $bytes = [System.IO.File]::ReadAllBytes((Join-Path $PSScriptRoot $relative))
                $entryStream.Write($bytes, 0, $bytes.Length)
            } finally { $entryStream.Dispose() }
        }
    } finally { $archive.Dispose() }
} finally { $stream.Dispose() }
$hash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
[System.IO.File]::WriteAllText($archivePath + '.sha256', "$hash  $([System.IO.Path]::GetFileName($archivePath))`n", [System.Text.UTF8Encoding]::new($false))
Write-Output $archivePath
Write-Output "SHA256 $hash"
