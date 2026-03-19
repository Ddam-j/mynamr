# mynamr

`mynamr`는 입력된 문자열을 기계적으로 분석하여, 내장된 규칙에 따라 재구성된 결과 문자열을 반환하는 CLI-first 텍스트 리네이밍 도구입니다.

## Development bootstrap

```bash
go test ./...
go run ./cmd/mynamr --version
go run ./cmd/mynamr rule list
```

## Release flow

- Release automation is configured with `.goreleaser.yaml`
- GitHub tag releases are handled by `.github/workflows/release.yml`
- Local tag helpers live in `scripts/release-tag.sh` and `scripts/release-tag.ps1`
- Detailed release instructions are in `docs/release.md`
