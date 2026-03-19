param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$ErrorActionPreference = 'Stop'

if ($Version -notmatch '^v\d+\.\d+\.\d+$') {
    throw 'Version must match vMAJOR.MINOR.PATCH'
}

$gitStatus = git status --porcelain
if ($gitStatus) {
    throw 'Working tree is dirty; commit or stash changes first'
}

git rev-parse $Version *> $null
if ($LASTEXITCODE -eq 0) {
    throw "Tag $Version already exists"
}

Write-Host "Running tests before tagging $Version..."
go test ./...
if ($LASTEXITCODE -ne 0) {
    throw 'go test failed'
}

Write-Host "Creating annotated tag $Version..."
git tag -a $Version -m "release: $Version"

Write-Host "Tag created locally: $Version"
Write-Host 'Next steps:'
Write-Host "  git push origin $Version"
Write-Host 'If the release commit is not on GitHub yet, push your branch first.'
Write-Host 'When the tag reaches GitHub, .github/workflows/release.yml will run GoReleaser.'
