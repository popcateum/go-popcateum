# v1.11.6 전환 클라이언트 운영 안내

기준 블록 8,904,999에서 이력을 이어 가는 운영 전환 설정이다. 비콘 genesis와 검증자 준비는 TTD 도달 전에 완료한다.

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
| London / Popstar | **`8,905,000`**, 해당 블록의 첫 거래부터 EIP-1559 및 예치 계약 사용 가능 |
| SeaPop | 실행 규칙 제거. 과거 DB 필드 `seapopBlock`은 이관 안전성 검사에만 사용 |
| TTD | **`7288759620794051`**, `terminalTotalDifficultyPassed=false` |
| Shanghai / Cancun / Prague | 예약 없음. v1.11.6은 Shanghai를 지원하나 CL과 일정을 맞춘 후 별도 활성화 |

Popstar부터 마지막 PoW 블록까지 난이도는 941,264다. 첫 London 블록의 base fee는 1 gwei이고 가스 한도 탄력성 규칙은 upstream EIP-1559를 따른다. 이후 기본 PoW 보상 30 POP + priority fee가 지급되며 base fee는 소각된다.

마지막 PoW 블록은 `parentTD < TTD <= blockTD`를 만족한다. 다음 블록은 비콘이 만드는 PoS 블록이어야 하며 PoW 보상은 없어진다. **난이도 폭탄을 추가하지 않는다.** 구버전 마이너가 별도 PoW 가지를 만들어도 업그레이드 노드가 terminal 블록의 PoW 후손을 정식 체인에 받아들이지 않는다. [EIP-3675](https://eips.ethereum.org/EIPS/eip-3675), [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559).

## 2. 머지 경계

기준 블록의 난이도 941,264를 8,905,000부터 8,999,999까지 95,000개 PoW 블록에 고정한다. 기준 TD에 해당 합을 더한 값이 운영 TTD다.

```text
7,288,670,200,714,051 + 95,000 × 941,264 = 7,288,759,620,794,051
```

gpop과 Prysm은 동일한 TTD를 사용한다. terminal 블록의 첫 자식이 PoS 높이 9,000,000이다. 해시파워는 이 경계에 도달하는 시각을 결정한다. Bellatrix 비콘과 활성 검증자가 준비되지 않으면 제안이 가능해질 때까지 다음 블록을 기다린다.

## 3. 빌드 및 기존 DB 이관

```sh
cd /Users/elixir/dev/project/popcateum/popcateum2-dev/go-popcateum
make gpop
./build/bin/gpop version
```

검증된 빌드 도구는 Go 1.20.14이다. Makefile은 Go 런처의 `GOTOOLCHAIN`을 지정한다. 직접 `go build`/`go test`를 사용할 때도 동일하게 지정한다. 전체 CLI는 `make all`로 빌드한다.

v1.11.6이 읽을 수 있는 원래 PoW DB를 사용할 때는 노드를 정상 종료한 뒤 전체 datadir 사본에서 시작한다. 최신 gpop이 작성한 v9 DB는 head가 8,904,999여도 v1.11.6과 ancient 메타데이터 형식이 다르다. 이 경우에는 원래 PoW 백업을 사용하거나, v9 호환 바이너리로 사본에서 정식 블록을 내보내고 새 디렉터리에 가져온다. 아래 export 대상은 처음 쓰는 파일이어야 한다(범위 export는 기존 파일에 이어 쓴다).

```sh
"$V9_GPOP" --datadir "$V9_COPY" export "$NEW_EXPORT_FILE" 0 8904999
./build/bin/gpop --datadir "$NEW_DATADIR" --syncmode full --gcmode archive import "$NEW_EXPORT_FILE"
```

가져오기 후 8,904,999의 해시·상태 루트·TD를 위 표와 대조한다. DB 버전 숫자나 freezer 메타데이터를 직접 수정하면 안 된다.

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

## 5. 예치와 Prysm 설정

Popstar는 거래 실행 전에 표준 예치 계약과 고정 대상 배치 계약을 설치한다. 초기 예치 트리와 배치 대상은 표준 생성자 실행 결과와 동일하다. 배포 거래나 생성 영수증은 없다. 첫 예치부터 표준 계약의 `DepositEvent`를 색인한다.

```yaml
# 운영 Prysm 체인 설정에 반영할 항목
DEPOSIT_CHAIN_ID: 1213
DEPOSIT_NETWORK_ID: 1213
DEPOSIT_CONTRACT_ADDRESS: "0x909ca70000000000000000000000000000000001"
TERMINAL_TOTAL_DIFFICULTY: "7288759620794051"
BASE_REWARD_FACTOR: 6400
```

Prysm의 `--contract-deployment-block 8905000`과 gpop의 인증 Engine API(기본 9551), 동일 JWT로 연결한다. 운영용 genesis fork version을 확정한 뒤 발행한 예치 데이터로 검증자를 준비한다. 배치 주소는 `0x909CA70000000000000000000000000000000002`다. 예치 페이지나 익스플로러가 생성 영수증을 요구한다면 포크 블록 해시·코드와 예치 이벤트로 검증하도록 연동해야 한다.

Merge 후 업그레이드 일정은 미정이다. 기존 리허설용 Prysm DB, genesis 및 별도 `local-merge`의 테스트 TTD는 이 메인넷 사양과 다르다.
