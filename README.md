# Clima por CEP - Go + Cloud Run

API que recebe um CEP (8 dígitos), identifica a cidade via ViaCEP e retorna temperatura atual em Celsius, Fahrenheit e Kelvin.

Como rodar localmente:

1. Build:

```sh
docker build -t weather-app .
```

2. Run (local, com `WEATHERAPI_KEY`):

```sh
# obrigatoriamente defina a chave via variável de ambiente
docker run -p 8080:8080 -e WEATHERAPI_KEY=SEU_KEY --name weather-app-local weather-app
```

Observação: a imagem não inclui fallback de chave `WEATHERAPI_KEY` é obrigatória.

Requisitos

- Go (para executar testes locais): `1.25.5`
- Docker (para o script `test/run-bdd.sh`)

Endpoints:

- `GET /weather?cep=01001000` -> retorna JSON com `temp_C`, `temp_F`, `temp_K`.

URL pública (Cloud Run):

https://weather-app-309527484550.us-central1.run.app

Testes

- Testes unitários (TDD):

	1. No diretório `src`, execute:

	```sh
	go test ./...
	```

	2. Para rodar somente um arquivo ou pacote específico, use o filtro do `go test`.

- Testes de componente (BDD):

	O projeto inclui um runner BDD que integra com `go test` e stubs via WireMock. As features estão em `test/features`.

	- Execução rápida (script): do diretório raiz do repositório execute:

		```sh
		chmod +x test/run-bdd.sh
		./test/run-bdd.sh
		```

		O script cria uma network Docker, sobe um container WireMock, builda e sobe a imagem da aplicação apontando para o WireMock e executa os testes BDD. A saída dos testes é salva em `test/test-output.log` e os logs dos containers são impressos ao final.

	- Executando manualmente:

		```sh
		# Linux / macOS (com socket /var/run/docker.sock)
		cd src
		go test ./... -v -run TestComponent
		```

		```powershell
		# Windows (PowerShell) - use o named pipe do Docker Desktop
		$env:DOCKER_HOST='npipe:////./pipe/docker_engine'
		cd src
		go test ./... -v -run TestComponent
		```

	Observação: o `godog` CLI não é necessário; o runner de testes está integrado como teste Go (`TestComponent`) e usa Dockertest + WireMock para isolar as APIs externas.

Execução local (resumo)

- Build da imagem:

```sh
docker build -t weather-app .
```

- Run local com `WEATHERAPI_KEY`:

```sh
docker run -p 8080:8080 -e WEATHERAPI_KEY=SEU_KEY --name weather-app-local weather-app
```

Observações

- A variável `WEATHERAPI_KEY` deve ser definida em ambientes de produção (Cloud Run) via secrets ou `--set-env-vars` no deploy. 
 - Os testes BDD usam as features em `test/features` e NÃO fazem chamadas a serviços externos durante a execução: os stubs do ViaCEP e do WeatherAPI são iniciados localmente pelo próprio runner de teste usando WireMock (via Dockertest). Ou seja, os testes de componente são determinísticos e isolados de terceiros.
	Para rodar os BDD localmente de forma automática use o script `test/run-bdd.sh` (requer Docker); se preferir rodar manualmente, veja a seção "Executando manualmente" acima — ela também usa os stubs iniciados pelo teste.

Deploy no Cloud Run (exemplo):

1. Autentique e selecione o projeto:

```sh
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
```

2. Build e enviar imagem para o Artifact Registry / Container Registry (exemplo usando Cloud Build):

```sh
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/weather-app:latest
```

3. Deploy no Cloud Run:

```sh
gcloud run deploy weather-app --image gcr.io/YOUR_PROJECT_ID/weather-app:latest --platform managed --region YOUR_REGION --allow-unauthenticated --set-env-vars WEATHERAPI_KEY=SEU_KEY
```

Substitua `YOUR_PROJECT_ID`, `YOUR_REGION` e `SEU_KEY` conforme necessário.
