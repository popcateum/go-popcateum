# Go Popcateum — PoS 전환 클라이언트

기존 Popcateum 이력을 유지하는 **Geth v1.11.6 기반 전환 버전**이다. PoW 채굴과 표준 Engine API를 지원하며 미래 TTD에서 Merge한다. SeaPop 합의 규칙은 제거했다.

**[빌드·기존 데이터 전환·채굴 실행 안내](docs/pos-transition.md)**를 먼저 읽는다. 원본 체인의 사본으로 먼저 검증한다. 비콘이 준비되지 않은 채 TTD에 도달하면 체인이 멈추므로 준비 기간에는 채굴을 중단할 수 있어야 한다.

| 항목 | 설정 |
|---|---|
| Chain ID / Network ID | `1213` / `1213` |
| 기존 이력 | genesis·PopSong(85,000)·Popcorn(4,000,000) 유지 |
| London | `8,905,000` |
| Terminal total difficulty | **`7288681830821503`** |
| 첫 PoS 높이 | `8,910,000` 부근을 목표로 추정; 고정 높이가 아님 |
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

## 검증 도구

```sh
GOTOOLCHAIN=go1.20.14 go test ./params ./consensus/... ./core/... ./miner ./eth/... ./cmd/gpop
python3 scripts/estimate_ttd.py
python3 scripts/smoke_transition.py
```

TTD 스크립트는 저장된 공개 헤더로 추정치를 재현한다. `--rpc http://127.0.0.1:9506 --continuous`는 재개 후 관측값으로 다시 계산만 한다. 설정을 자동 변경하지 않는다. smoke 스크립트는 공개 테스트 키와 임시 체인·소형 테스트 DAG를 사용하며 실제 비콘 합의를 실행하지 않는다.

upstream: [Geth v1.11.6](https://github.com/ethereum/go-ethereum/releases/tag/v1.11.6), commit `ea9e62ca3db5c33aa7438ebf39c189afd53c6bf8`. 라이브러리 LGPL-3.0, `cmd` 실행 프로그램 GPL-3.0 등 기존 라이선스와 저작권 표기를 유지한다.
