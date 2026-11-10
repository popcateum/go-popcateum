# v1.11.6 전환 클라이언트 운영 안내

작성: 2026-11-10 KST. 이번 구현은 **기존 PoW 체인 재개·예치 준비 및 향후 TTD Merge를 위한 EL**이다. 실제 예치 계약·비콘 genesis·validator 설정은 다음 단계에서 확정한다. 전체 과거 트랜잭션의 재실행과 실제 비콘 finality 리허설을 완료했다는 의미는 아니다.

## 1. 보존한 것과 새 일정

| 항목 | 값 / 동작 |
|---|---|
| 기존 Popcateum Git 기준 | `1f0363dc68395d2de8a11da9404fdfea24d14225` |
| upstream | Geth v1.11.6 `ea9e62ca3db5c33aa7438ebf39c189afd53c6bf8` |
| genesis | `0x07fd751edea3b3a2d790002e4712b14e2d0e43670f25952d4734ee31b526fa1b` |
| chain / network ID | `1213` / `1213` |
| 검증 기준 head | `8,904,999`, `0x185e96f351cdaf980c707b80b37097803d092bf36f9841ad384ea5b4dba4ab00` |
| 기준 state root | `0x34b8409b1b43c7491316d3a090c0dbfb45824a48865c5bb8f724aafb9b919e26` |
| 기준 total difficulty | `7288670200714051` |
| 기존 실행 포크 | Berlin까지 기존 설정 그대로 유지 |
| PopSong | `85,000`: 6초 기준 조정, 기존 12M bomb delay 유지 |
| Popcorn | `4,000,000`: 기본 PoW 보상 50 → 30 POP, uncle 규칙 유지 |
| London | **`8,905,000`**, 다음 블록부터 EIP-1559 적용 |
| SeaPop | 실행 규칙 제거. 과거 DB 필드 `seapopBlock`은 이관 안전성 검사에만 사용 |
| TTD | **`7288681830821503`**, `terminalTotalDifficultyPassed=false` |
| Shanghai / Cancun / Prague | 예약 없음. v1.11.6은 Shanghai를 지원하나 CL과 일정을 맞춘 후 별도 활성화 |

London에서도 PopSong 난이도 조정식을 유지한다. Ethereum London의 난이도 폭탄 일정으로 교체하지 않는다. 첫 London 블록의 base fee는 1 gwei이고 가스 한도 탄력성 규칙은 upstream EIP-1559를 따른다. 이후 기본 PoW 보상 30 POP + priority fee가 지급되며 base fee는 소각된다.

마지막 PoW 블록은 `parentTD < TTD <= blockTD`를 만족한다. 다음 블록은 비콘이 만드는 PoS 블록이어야 하며 PoW 보상은 없어진다. **난이도 폭탄을 추가하지 않는다.** 구버전 마이너가 별도 PoW 가지를 만들어도 업그레이드 노드가 terminal 블록의 PoW 후손을 정식 체인에 받아들이지 않는다. [EIP-3675](https://eips.ethereum.org/EIPS/eip-3675), [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559).

## 2. TTD 추정 근거와 준비 시간

[공개 샘플](evidence/ttd-headers.json)은 8,904,487~8,904,999의 513개 헤더, 즉 512개 간격이다. 평균 간격 1.751953125초, 평균 난이도(정수) 837,696, 마지막 난이도 941,264였다. 과거 해시파워 추정치는 약 478,150이며 앞으로도 같다고 가정했다.

고정 난이도를 곱하는 대신 PopSong 조정식으로 미래 5,000개 PoW 블록을 128회 모의 계산했다. 장기 정지 뒤 첫 블록의 난이도 감소, 이후 난이도 상승, 초 단위 타임스탬프를 반영했다. seed는 20261110이다.

```text
예상 마지막 PoW 높이 = 8,904,999 + 5,000 = 8,909,999
예상 추가 TD 중앙값 = 11,630,107,452
TTD = 7,288,670,200,714,051 + 11,630,107,452
    = 7,288,681,830,821,503
```

[재현 결과](evidence/ttd-estimate.json)의 순수 가동 시간 중앙값은 약 **6.37시간**이다. 재시작 정지 시간, DAG 생성, 재조직, 미래 장비·해시파워 변화, 예치·비콘 준비 시간을 예측한 값이 아니다. 모의 계산 분위수도 실제 미래의 신뢰구간이 아니다. 실제 첫 PoS 높이는 8,910,000과 달라질 수 있다.

```sh
# 포함된 공개 샘플로 현재 바이너리에 넣은 값 재현
python3 scripts/estimate_ttd.py
# 채굴 재개 후: 첫 블록 장기 정지 보정을 다시 적용하지 않음
python3 scripts/estimate_ttd.py --rpc http://127.0.0.1:9506 --continuous
# 해시파워 가정의 영향 확인
python3 scripts/estimate_ttd.py --hashrate-scale 2
```

**높이만 보지 말고 TD를 관측한다.** 계약 배포·예치 확인·비콘 genesis 대기와 fork 활성화는 이 시간보다 오래 걸릴 수 있다. 준비 중에는 `miner.stop()` 또는 외부 마이너 중단으로 TTD 도달을 늦춘다. 다른 마이너가 있으면 운영 합의가 필요하다. 비콘이 준비되지 않았는데 TTD를 넘으면 새 클라이언트는 다음 PoW 블록을 만들지 않는다. TTD 변경이 필요하면 **도달 전에 모든 EL/CL이 같은 값으로 재배포**돼야 하며 자동으로 연장하지 않는다.

## 3. 빌드 및 기존 DB 이관

```sh
cd /Users/elixir/dev/project/popcateum/popcateum2-dev/go-popcateum
make gpop
./build/bin/gpop version
```

검증된 빌드 도구는 Go 1.20.14이다. Makefile은 Go 런처의 `GOTOOLCHAIN`을 지정한다. 직접 `go build`/`go test`를 사용할 때도 동일하게 지정한다. 전체 CLI는 `make all`로 빌드한다.

실행 중인 기존 노드는 정상 종료 후 전체 datadir을 백업한다. **실행 중인 LevelDB를 파일 복사하지 않는다.** 이 안내를 작성하면서 원본 `/Users/elixir/dev/project/popcateum/gpop/chain/Popcateum` DB를 새 바이너리로 열지 않았다. 원본 파일에 쓴 내용도 없다.

1. 원본과 별도로 만든 완전한 사본을 새 바이너리로 먼저 연다. `gpop/chaindata`와 `ancient`를 함께 보존하며, 별도 ancient 경로를 쓰면 그것도 사본으로 지정한다.
2. 기존 DB에 `init`을 다시 하지 않는다. Mainnet genesis를 인식하면 호환성 확인 후 새 London/TTD 설정을 기록한다.
3. DB의 기존 `seapopBlock`에 이미 도달했다면 이관을 거부한다. 현재 기준 8,904,999에서는 아직 실행되지 않은 포크이므로 제거한다.
4. DB 스키마 갱신 후 구버전으로 같은 사본을 다시 열지 않는다. 롤백은 보존한 원본/백업에서 한다.
5. 먼저 채굴 없이 head·hash·stateRoot·기존 잔액·컨트랙트 호출·로그 조회를 대조한다. 이관 중 예상하지 않은 rewind가 있으면 채굴을 시작하지 말고 로그를 조사한다.

## 4. 실행과 채굴

아래 `DATADIR`은 **검증용 사본 경로**, `MINER`는 본인의 실제 보상 주소로 바꾼다. 기본 HTTP/WS 포트는 9506/9507, Engine은 9551, P2P는 TCP/UDP 60606이다. 기존 노드를 함께 띄우면 포트도 분리해야 한다.

```sh
DATADIR="/absolute/path/to/Popcateum-copy"
MINER="0xYOUR_MINING_REWARD_ADDRESS"
./build/bin/gpop --datadir "$DATADIR" \
  --syncmode full --gcmode archive \
  --miner.etherbase "$MINER" \
  --http --http.addr 127.0.0.1 --http.api "web3,eth,txpool,ethash,net,debug" \
  --ws --ws.addr 127.0.0.1 --ws.api "web3,eth,txpool,ethash,net,debug" \
  --authrpc.addr 127.0.0.1 --authrpc.port 9551 console
```

```javascript
eth.blockNumber
eth.getBlock(0).hash
eth.getBlock(8904999).hash
eth.getBlock(8904999).stateRoot
eth.getBlock("latest").totalDifficulty.toString(10)
admin.nodeInfo.protocols.eth.config

// 위 대조 완료 후 시작. 1은 CPU 채굴 스레드 수이다.
miner.start(1)
// 계약/비콘 준비 중 TTD까지의 여유가 부족하면 중단한다.
miner.stop()
```

GPU/외부 마이너는 기존 `ethash` RPC와 보상 주소를 사용한다. CLI `--miner.popcatbase`와 RPC/콘솔 `miner.setPopcatbase`는 호환 alias로 남겨뒀다. 표준 `--miner.etherbase`, `miner.setEtherbase`, `eth_coinbase`, `eth_etherbase`도 지원한다. PoW 보상 주소 지정은 그 주소의 개인키 unlock을 요구하지 않는다. 트랜잭션 서명은 별도 지갑으로 한다. `personal` HTTP API는 위 명령에 노출하지 않는다.

첫 채굴 시 해당 epoch DAG 생성 때문에 기다릴 수 있다. 이 시점에 필요한 실제 DAG 용량·채굴 성능은 사용 장비에서 확인한다. `--fakepow`나 smoke의 `PowMode=2`는 운영에 사용하지 않는다.

## 5. 계약과 비콘 연결의 다음 단계

표준 deposit contract를 **실제 배포한 주소**로 사용한다. 각 32 POP에 대응하는 BLS pubkey·withdrawal credentials·서명·deposit data root가 정확해야 한다. 하나의 운영 지갑에서 여러 validator의 예치를 보낼 수 있다. 최종 유효 잔액·활성화는 CL 규칙에 따라 처리한다.

Shanghai가 아직 비활성화이므로 새 계약 컴파일의 EVM target은 **London 이하**로 지정한다. 최신 컴파일러 기본 bytecode의 `PUSH0` 등 Shanghai 명령을 사용하면 배포/실행에 실패할 수 있다. 실제 표준 deposit bytecode, ABI, 배포 receipt·block/hash, deposit root/count·이벤트를 확인한 뒤 예치 페이지를 구현한다.

EL은 Engine API를 제공한다. JWT 기본 파일은 `<datadir>/gpop/jwtsecret`이며 CL이 같은 값을 사용해야 한다. 원격 CL이면 JWT 파일을 안전하게 전달하고 통신·허용 host를 명시한다. 운영 CL config에는 동일 TTD **7288681830821503**, chain/network 1213, 실제 deposit 주소·배포 블록·네트워크별 fork version/genesis 정보를 넣는다. deposit 주소는 아직 없으므로 이 클라이언트에 임의 주소를 넣지 않았다.

현재 Lighthouse/Prysm 최신 버전이 과거 PoW→Merge bootstrap을 그대로 지원한다고 가정하지 않는다. 공식 과거 release 중 pre-Merge genesis/Bellatrix/Engine V1을 지원하는 조합으로 리허설하고, **비콘 genesis 및 Bellatrix가 준비된 상태에서 TTD에 도달**하게 한다. Shanghai timestamp는 CL Capella epoch와 이후 별도 예약한다. Cancun/Deneb, Prague/Electra는 더 새 Geth 기반 업그레이드가 필요하다.

## 6. 검증과 한계

전체 도구 빌드·전체 Go 패키지 테스트·린트 결과는 [검증 기록](evidence/validation.json)에 저장했다. 외부 `tests/testdata`, `tests/evm-benchmarks` submodule은 초기화하지 않았으므로 해당 외부 fixture를 요구하는 테스트는 upstream 규칙에 따라 skip된다.

- 원래 genesis hash 유지, 과거 PopSong/Popcorn 경계 및 기준 head의 실제 Ethash seal 검증.
- PoW 보상·uncle 보상, SeaPop 예정 높이를 지나도 30 POP 유지.
- 기존 설정을 London/TTD로 이관하고 head 유지, 이미 SeaPop이 적용된 DB 거부.
- 첫 London 헤더의 base fee·가스 한도·장기 정지 후 난이도 검증.
- TTD 전후 PoS/PoW 헤더 판정, Merge 이후 EL PoW 보상 중단.
- `scripts/smoke_transition.py`: 임시 체인에서 실제 CLI, 소형 테스트 DAG의 채굴, 계약 배포, London type-2 거래, TTD 도달 후 채굴 정지. 운영 해시파워/전체 DAG 또는 실제 CL 네트워크 검증은 아니다.

실제 datadir 사본에서의 전 구간 state 재실행, 실제 비콘·validator finality, 실제 예치 계약 배포·일괄 예치, 다중 서버 장애 복구는 후속 리허설 항목이다. 이 버전은 요청한 역사적 v1.11.6 기반 전환 버전이며 최신 Ethereum 하드포크/수정 전체를 포함한 최신 Geth라는 의미는 아니다.
