#!/usr/bin/env sh
set -euo pipefail

# Script para subir WireMock, buildar/rodar a app em container e executar os testes BDD
# Uso: ./test/run-bdd.sh (execute na raiz do repositório)

REPO_ROOT="$(pwd)"
NETWORK=weather-net
WIREMOCK_NAME=wiremock-bdd
APP_NAME=weather-app-bdd
APP_IMAGE=weather-app:local

echo "-> criando network $NETWORK (se não existir)"
docker network inspect "$NETWORK" >/dev/null 2>&1 || docker network create "$NETWORK"

echo "-> iniciando WireMock em container ($WIREMOCK_NAME)"
# remover container antigo com mesmo nome, se existir
docker rm -f "$WIREMOCK_NAME" >/dev/null 2>&1 || true
docker run -d --name "$WIREMOCK_NAME" --network "$NETWORK" -p 8081:8080 wiremock/wiremock:2.35.0

echo "-> buildando imagem da app: $APP_IMAGE"
docker build -t "$APP_IMAGE" .

echo "-> iniciando container da app ($APP_NAME) apontando para o WireMock"
# remover container antigo da app, se existir
docker rm -f "$APP_NAME" >/dev/null 2>&1 || true
  docker run -d --name "$APP_NAME" --network "$NETWORK" -p 8080:8080 \
  -e VIACEP_BASE_URL="http://$WIREMOCK_NAME:8080" \
  -e WEATHERAPI_BASE_URL="http://$WIREMOCK_NAME:8080" \
  -e WEATHERAPI_KEY="testkey" \
  "$APP_IMAGE"

echo "-> aguardando WireMock ficar pronto"
sleep 2
retry=0
until [ $retry -ge 15 ]
do
  if curl -sS http://localhost:8081/__admin/mappings >/dev/null 2>&1; then
    break
  fi
  retry=$((retry+1))
  sleep 1
done

if [ $retry -ge 15 ]; then
  echo "WireMock não respondeu a tempo" >&2
  exit 1
fi

echo "-> executando testes BDD"

# arquivo para capturar saída dos testes
TEST_LOG="$REPO_ROOT/test/test-output.log"
rm -f "$TEST_LOG"

# Detecta se o socket unix do Docker está disponível (Linux/macOS)
if [ -S /var/run/docker.sock ]; then
  echo "-> executando testes dentro de um container Go (socket unix disponível)"
  set +e
  docker run --rm \
    --network "$NETWORK" \
    -v "$REPO_ROOT/src":/src \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -w /src \
    golang:1.20 sh -c 'go test ./... -v -run TestComponent' 2>&1 | tee "$TEST_LOG"
  EXIT_CODE=${PIPESTATUS[0]:-$?}
  set -e
else
  echo "-> socket unix do Docker não encontrado. Detectando ambiente..."
  UNAME=$(uname | tr '[:upper:]' '[:lower:]' || true)
  if echo "$UNAME" | grep -qi "mingw\|msys\|cygwin"; then
    echo "-> Ambiente Windows detectado (Git Bash). Ajustando DOCKER_HOST para named pipe e executando testes localmente."
    export DOCKER_HOST='npipe:////./pipe/docker_engine'
  fi

  echo "-> executando 'go test' no host (diretório: $REPO_ROOT/src)"
  set +e
  (cd "$REPO_ROOT/src" && go test ./... -v -run TestComponent) 2>&1 | tee "$TEST_LOG"
  EXIT_CODE=${PIPESTATUS[0]:-$?}
  set -e
fi

echo "-> limpando containers"
docker rm -f "$APP_NAME" >/dev/null 2>&1 || true
docker rm -f "$WIREMOCK_NAME" >/dev/null 2>&1 || true
docker network rm "$NETWORK" >/dev/null 2>&1 || true

echo
echo "================== TEST OUTPUT =================="
if [ -f "$TEST_LOG" ]; then
  cat "$TEST_LOG"
else
  echo "(nenhum arquivo de log de teste encontrado)"
fi

echo
echo "================== CONTAINER LOGS: $APP_NAME =================="
docker logs "$APP_NAME" 2>&1 || echo "(nenhum log do container $APP_NAME)"

echo
echo "================== CONTAINER LOGS: $WIREMOCK_NAME =================="
docker logs "$WIREMOCK_NAME" 2>&1 || echo "(nenhum log do container $WIREMOCK_NAME)"

exit $EXIT_CODE
