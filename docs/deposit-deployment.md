# PoW 상태에서 표준 예치 컨트랙트 배포

> **Popstar 메인넷:** 표준 계약(끝자리 01)과 배치 계약(끝자리 02)은 8,905,000에서 자동 설치된다. 아래 수동 배포 방법은 별도 계약을 사용하는 개발 체인용이다. 운영 주소와 Prysm 설정은 [운영 전환 문서](pos-transition.md)를 따른다.


작성: **2026-11-11 KST**. 이번 준비는 운영자가 직접 배포할 수 있는 도구·산출물·명령이다. 실제 Popcateum 체인에 계약을 배포하거나 실제 지갑으로 서명하지 않았다.

## 배포할 계약

Ethereum [consensus-specs v1.3.0의 표준 DepositContract](https://github.com/ethereum/consensus-specs/tree/01b53691dcc36d37a5ad8994b3a32d8de69fb1aa/solidity_deposit_contract)를 수정 없이 사용한다. 기존 `fork-script`의 ABI/배포 bytecode도 이 공식 산출물과 동일함을 확인했다. 과거 배포 주소와 예치 데이터는 이번 배포에 재사용하지 않는다.

- 계약 소스: [deposit_contract.sol](../contracts/deposit/deposit_contract.sol), 배포 ABI/bytecode: [deposit_contract.json](../contracts/deposit/deposit_contract.json).
- 고정 upstream commit: `01b53691dcc36d37a5ad8994b3a32d8de69fb1aa`. [출처·SHA-256](../contracts/deposit/provenance.json).
- Solidity 0.6.11로 빌드된 공식 bytecode이며 Shanghai의 `PUSH0`를 요구하지 않는다. Berlin·London 실행 환경에서 모두 검증했다. 운영자가 solc/npm을 설치하거나 재컴파일할 필요가 없다.
- 생성자 인자·관리자·proxy·업그레이드 기능이 없다. **배포할 때 전송하는 POP는 0이고 가스 비용만 필요하다.** 32 POP 예치는 다음 단계다.
- 네트워크 전체가 사용할 계약 **하나**를 배포한다. validator마다 새 계약을 만드는 방식이 아니다.
- 표준 ABI·이벤트·예치 트리를 유지하므로 이후 stock CL이 실제 주소와 배포 블록을 설정으로 받아 처리할 수 있다.

| 검증 항목 | 기대값 |
|---|---|
| Chain ID / Network ID | `1213` / `1213` |
| genesis | `0x07fd751edea3b3a2d790002e4712b14e2d0e43670f25952d4734ee31b526fa1b` |
| creation code Keccak-256 | `0x282630916603531e25a075a4fe21c2b612a5010591c30b4342598de1f4fc4e09` |
| runtime code Keccak-256 | `0x6c029a231254fadb724d63be769f75eedd66362df034a3e663252b49d062a666` |
| 초기 deposit count | `0` |
| 초기 deposit root | `0xd70a234731285c6804c2a4f56711ddb8c82c99740f207854891028af34e27e5e` |

Solidity의 `ether`는 네이티브 코인 단위 `10^18`을 뜻하므로 이 체인에서는 POP이다. 소스의 `1 ether`를 문자열 치환할 필요가 없다. 표준 계약의 최소 입금은 1 POP, 단위는 1 gwei이며 **validator 활성화 기준 32 POP은 CL/예치 도구 정책**이다. 표준 계약은 BLS 서명을 검사하지 않는다. 배포 성공·DepositEvent 발생만으로 validator 등록 성공을 판단하지 않는다. [공식 예치 명세](https://github.com/ethereum/consensus-specs/blob/v1.3.0/specs/phase0/deposit-contract.md).

## 1. 배포 도구 빌드

```bash
cd /Users/elixir/dev/project/popcateum/popcateum2-dev/go-popcateum
make depositctl
./build/bin/depositctl plan --help
```

Go 1.20.14 선택은 기존 Makefile과 같다. `make all`에도 포함된다. 합의 설정이나 현재 gpop DB를 변경하는 도구가 아니다.

## 2. 노드와 배포 지갑 준비

기존 [실행 안내](pos-transition.md)대로 새 gpop을 실행하고, 기존 head/state 및 사본 이관 검증을 마친 노드의 RPC를 사용한다. 도구는 HTTP 또는 IPC로 접속한다.

```bash
RPC="http://127.0.0.1:9506"
DEPLOYER="0x여기에_잔액이_있는_배포지갑_주소"
KEYSTORE="/절대경로/배포지갑의_UTC--keystore파일"
```

`KEYSTORE`는 **일반 실행 지갑의 암호화된 keystore JSON**이다. BLS validator keystore나 mnemonic 파일을 넣지 않는다. 기존 gpop keystore는 기본적으로 `<datadir>/keystore/UTC--...`에 있다. 주소 목록이 필요하면 다음 명령으로 확인한다.

```bash
./build/bin/gpop --datadir "/Users/elixir/dev/project/popcateum/gpop/chain/Popcateum" account list
```

- 해당 지갑에 배포 수수료를 낼 POP가 있어야 한다. 새 지갑이면 먼저 POP를 보낸다. 배포 지갑과 채굴 보상 지갑은 달라도 된다.
- `eth,net,web3` RPC면 충분하다. 공개 RPC에 `personal`, `admin`, `miner`를 추가하거나 계정을 unlock할 필요가 없다. 도구가 로컬에서 복호화·서명하고 `eth_sendRawTransaction`으로 보낸다.
- Engine API `9551`이 아니라 일반 RPC `9506` 또는 IPC를 지정한다. IPC 예: `RPC="/절대경로/Popcateum/gpop.ipc"`.
- 배포에는 PoW 블록 생성이 필요하다. 채굴은 운영자가 콘솔에서 직접 시작·중단한다. 도구가 채굴을 자동 시작하지 않는다.

## 3. 전송하지 않고 배포 계획 생성

```bash
./build/bin/depositctl plan \
  --rpc "$RPC" \
  --from "$DEPLOYER" \
  --gas-price-gwei 2 \
  --out deployments/deposit-plan.json
```

이 명령은 지갑·체인을 변경하지 않는다. 다음을 확인한다.

1. 실제 chain/network ID와 genesis가 Popcateum인지.
2. 노드가 syncing 중이 아니고, PoW head의 TD가 이번 클라이언트 TTD `7288759620794051` 미만인지.
3. 배포 지갑이 EOA이고 처리 중인 nonce가 없는지, 최대 수수료를 낼 잔액이 있는지.
4. 공식 생성 코드의 gas estimate와 20%+10,000 gas 여유가 현재 block gas limit 안에 들어가는지.
5. London 상태라면 지정 gas price가 현재 base fee 이상인지.

`deployments/deposit-plan.json`의 `from`, `nonce`, `expected_contract_address`, `gas_limit`, `gas_price_wei`, **`maximum_fee_pop`**, `value_wei: "0"`, `observed_total_difficulty`를 확인한다. gas 가격은 **정수 gwei**로 지정한다. 도구는 EIP-155로 보호된 legacy 거래를 사용하므로 London 전후에 배포할 수 있다.

로컬 테스트의 실제 배포 gas는 **2,146,480**이었다. gas price 2 gwei라면 실제 수수료는 약 **0.00429296 POP**, 계획상의 여유를 포함한 최대 수수료는 약 **0.005171552 POP**이다. 실제 계획 결과를 우선하며 가스 가격·블록 조건에 따라 달라질 수 있다. 사용하지 않은 gas에는 비용이 청구되지 않는다.

파일은 덮어쓰지 않는다. nonce·수수료를 재산정할 때는 `deposit-plan-v2.json`처럼 새 파일명을 사용한다. 서로 다른 계획을 섞지 않는다.

## 4. 채굴 준비 후 실제 배포

채굴이 멈춰 있다면 **gpop의 로컬 콘솔**에서 실제 보상 주소를 지정하고 시작한다.

```javascript
miner.setEtherbase("0x여기에_실제_채굴보상주소")
miner.start(1)
```

그 후 별도 터미널에서 실행한다. **아래 `send`가 실제 서명·전송을 수행하는 명령이다.**

```bash
./build/bin/depositctl send \
  --rpc "$RPC" \
  --plan deployments/deposit-plan.json \
  --keystore "$KEYSTORE" \
  --confirmations 3 \
  --timeout 5m \
  --out deployments/deposit-mainnet.json
```

암호를 숨김 입력으로 요청한다. 암호나 개인키를 명령 인자에 직접 넣지 않는다. 자동화가 필요할 때만 권한을 제한한 기존 비밀번호 파일을 `--password-file /절대경로/password-file`로 지정한다. 인터랙티브 암호 입력 시 출력 redirect/pipe를 하지 않는다. JSON 결과는 `--out` 파일에 저장된다.

전송 직전 nonce·잔액·가스 조건·PoW/TTD 상태를 다시 확인한다. 계획 작성 이후 같은 지갑에서 다른 거래를 보냈다면 새 계획이 필요하다. 계획의 관측 블록이 reorg된 경우도 재계획한다.

도구는 **전송 전에 트랜잭션 해시와 예상 주소를 기록**하고, receipt·3개 confirmation·정확한 runtime bytecode·deposit root/count를 확인한 뒤 `phase: "verified"`로 갱신한다. confirmation 수에는 배포 블록 자신이 포함된다. 이는 비콘의 deposit follow distance나 PoS finality를 대신하지 않는다.

배포와 필요한 확인이 끝나고 다음 준비에 시간이 걸리면 콘솔에서 채굴을 중단한다.

```javascript
miner.stop()
eth.getBlock("latest").totalDifficulty.toString(10)
```

**TTD는 준비 완료를 기다려 주지 않는다.** 비콘·validator를 준비하기 전에 TTD에 도달하지 않도록 관측한다. 다른 채굴자가 있으면 그 채굴도 별도로 조율해야 한다.

## 5. 재검증과 전송 중단 복구

응답 오류나 timeout이 나도 계약이 이미 배포됐거나 거래가 pending일 수 있다. **새 계약을 바로 다시 배포하지 않는다.** `--out` journal의 `transaction_hash`로 먼저 확인한다. `prepared`도 RPC 응답이 끊긴 상황에서는 전송 실패를 확정하는 뜻이 아니다.

```bash
TX=$(python3 -c 'import json; print(json.load(open("deployments/deposit-mainnet.json"))["transaction_hash"])')

./build/bin/depositctl verify \
  --rpc "$RPC" \
  --tx "$TX" \
  --confirmations 3 \
  --out deployments/deposit-rechecked.json
```

`verify`는 개인키·암호를 요구하지 않고 전송도 하지 않는다. pending이면 채굴과 txpool 상태를 확인한 뒤 다시 실행한다. receipt 실패면 원인을 조사한다. nonce가 이미 소비되었는지 확인하기 전에는 새로운 `send`를 실행하지 않는다. 전송 실패 이후 도구가 수수료 인상·nonce 교체를 자동 수행하지 않는다.

검증 성공 시 다음 정보를 보관한다.

- `deposit_contract_address`, `deployment_block`, `deployment_block_hash`, `transaction_hash`.
- `chain_id`, `genesis_hash`, `runtime_code_keccak256`.
- `deposit_root`, `deposit_count`, 해당 조회 시점의 `observed_block/hash`.

예치가 이미 시작됐다면 root/count는 초기값과 다를 수 있다. 도구는 조회 시점의 값을 기록한다. receipt와 조회 블록의 canonical hash도 확인한다. 이후 reorg가 발생할 수 있으므로 비콘 설정을 배포하기 전 다시 검증한다.

## 6. 비콘 및 예치 페이지로 전달

검증된 manifest에서 아래 값을 가져간다. 이 YAML은 **공통 예치 설정 일부**이며 완성된 CL config가 아니다.

```yaml
DEPOSIT_CHAIN_ID: 1213
DEPOSIT_NETWORK_ID: 1213
DEPOSIT_CONTRACT_ADDRESS: "실제 deposit_contract_address"
```

`deployment_block`은 선택 CL release의 deposit 시작 블록 설정/`deploy_block.txt` 등에 반영한다. 실제 release의 형식은 별도 검증한다. 최초 조회는 배포 블록을 포함해야 하며 기존 임시 계약 주소나 Ethereum mainnet deposit 주소를 넣지 않는다. 기존 블록 이력의 새 genesis를 만드는 작업도 아니다.

이후 작업 순서:

1. 이 실제 주소·ABI·chain 1213을 사용하는 예치 페이지 및 한 지갑의 다중 validator 예치 도구 작성.
2. Popcateum의 `GENESIS_FORK_VERSION`·withdrawal credentials/주소·validator 키 정책 확정.
3. 각 32 POP의 BLS 서명·SSZ deposit data root·네트워크 도메인 검증 후 실제 예치.
4. 실제 DepositEvent·count/root·인덱스로 비콘 genesis 구성, Bellatrix/Engine/JWT/TTD 일치 검증.

배포 지갑이 자동으로 validator가 되거나 출금 권한을 갖는 것은 아니다. 계약에는 일반적인 `withdraw()`가 없으며 CL의 유효한 validator·출금 경로가 필요하다. 아직 비콘 fork version을 확정하지 않았으므로 이 도구는 **실제 validator 예치 데이터를 생성하거나 예치금을 보내지 않는다**.

## 재현 테스트

```bash
make gpop depositctl
GPOP_BINARY="$PWD/build/bin/gpop" GOTOOLCHAIN=go1.20.14 \
  go test ./contracts/deposit ./cmd/depositctl -count=1 -v
make lint
```

계약 테스트는 Berlin/London 배포, 초기 tree, 32 POP 두 건의 이벤트·root/count, 잘못된 금액·root·pubkey 길이 및 단순 송금 거부를 검증한다. CLI 테스트는 별도 datadir·작은 Ethash 테스트 DAG·공개 테스트 키를 사용하여 PoW에서 계획→서명→receipt→검증, stale nonce 거부, timeout journal 복구를 확인한다. 테스트 바이너리 안에서만 synthetic genesis를 허용한다. **배포용 binary에는 임의 genesis 허용 옵션이 없다.**

실제 Popcateum 배포·실제 개인키 사용·실제 BLS validator 등록·CL finality는 이 테스트 범위에 포함되지 않는다.

## 일괄 예치 페이지 연결

표준 계약 배포 후 `depositctl plan --batch-target <표준계약주소>`로 별도 BatchDeposit을 배포할 수 있다. [페이지 실행·배포 안내](../../deposit-page/README.md)에 단일 거래 일괄 예치와 설정 생성 명령을 정리했다. 비콘의 deposit 주소는 계속 표준 계약 주소를 사용한다. GENESIS_FORK_VERSION 확정 전에는 페이지 전송을 차단한다.
