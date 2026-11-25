# Go Popcateum — PoS 전환 클라이언트

기존 Popcateum 이력을 유지하는 **Geth v1.11.6 기반 전환 버전**이다. PoW 채굴과 표준 Engine API를 지원하며 미래 TTD에서 Merge한다. SeaPop 합의 규칙은 제거했다.

**[빌드·기존 데이터 전환·채굴 실행 안내](docs/pos-transition.md)**를 먼저 읽는다. 원본 체인의 사본으로 먼저 검증한다. 비콘이 준비되지 않은 채 TTD에 도달하면 체인이 멈추므로 준비 기간에는 채굴을 중단할 수 있어야 한다.

| 항목 | 설정 |
|---|---|
| Chain ID / Network ID | `1213` / `1213` |
| 기존 이력 | genesis·PopSong(85,000)·Popcorn(4,000,000) 유지 |
| London · Popstar | `8,905,000` (두 예치 계약 설치 및 PoW 난이도 941,264 적용) |
| Terminal total difficulty | **`7288759620794051`** |
| 첫 PoS 높이 | `9,000,000` (마지막 PoW `8,999,999`) |
| Shanghai | 아직 예약하지 않음; Merge 후 CL Capella와 함께 예약 |
| 기본 포트 | HTTP `9506`, WS `9507`, Engine `9551`, P2P `60606` |
| IPC / DB | `gpop.ipc` / `<datadir>/gpop/chaindata` |

## 빌드

C 컴파일러와 Go가 필요하다. 최신 Go 런처에서는 Makefile이 **Go 1.20.14**를 선택·다운로드한다. 자동 선택을 지원하지 않는 구형 Go 설치에서는 Go 1.20.14를 직접 설치한다. 이 버전의 upstream 의존성은 최신 Go로 바로 링크되지 않을 수 있다.

```sh
make gpop
./build/bin/gpop version
```

전체 도구는 `make all`로 빌드한다. 이 저장소의 바이너리는 `gpop`이며 Ethereum 배포 사이트의 `geth` 바이너리로 대체하지 않는다.

## Popstar 예치 계약

| 역할 | 주소 |
|---|---|
| Prysm이 추적하는 표준 예치 계약 | `0x909ca70000000000000000000000000000000001` |
| 32 POP 일괄 예치 계약 | `0x909CA70000000000000000000000000000000002` |

업그레이드한 마이너가 블록 8,905,000을 채굴하면 일반 거래 실행 전에 두 계약이 설치된다. Prysm의 예치 이벤트 조회 시작 높이도 8,905,000이다.

## 검증 도구

```sh
GOTOOLCHAIN=go1.20.14 go test ./params ./consensus/... ./core/... ./miner ./eth/... ./cmd/gpop
python3 scripts/smoke_transition.py
```

smoke 스크립트는 공개 테스트 키와 임시 체인·소형 테스트 DAG를 사용하며 실제 비콘 합의를 실행하지 않는다.

upstream: [Geth v1.11.6](https://github.com/ethereum/go-ethereum/releases/tag/v1.11.6), commit `ea9e62ca3db5c33aa7438ebf39c189afd53c6bf8`. 라이브러리 LGPL-3.0, `cmd` 실행 프로그램 GPL-3.0 등 기존 라이선스와 저작권 표기를 유지한다.
