# Client-Server API — Cotação USD-BRL

Desafio Go: dois processos (`server.go` + `client.go`) trocando cotação dólar/real com timeouts estritos via `context`.

## Estrutura

```
client-server-api/
├── go.mod
├── server/
│   └── server.go        # HTTP server na :8080, endpoint /cotacao
├── client/
│   └── client.go        # CLI que consome /cotacao e grava prices.txt
└── prices.db            # SQLite criado em runtime (server)
```

## Requisitos atendidos

### Server (`:8080/cotacao`)
- Consome `https://economia.awesomeapi.com.br/json/last/USD-BRL` com timeout de **200ms**.
- Persiste cada cotação em SQLite (tabela `prices`) com timeout de **10ms**.
- Retorna `{"bid": "<valor>"}` em JSON.
- Loga no console quando qualquer um dos timeouts é excedido.

### Client
- Faz `GET http://localhost:8080/cotacao` com timeout de **300ms**.
- Extrai apenas o campo `bid` da resposta.
- Grava em `prices.txt` no formato `Dólar: {valor}`.
- Loga erro no console quando o timeout é excedido.

## Como rodar

Em terminais separados:

```bash
# Terminal 1 — sobe o servidor
cd server
go run server.go

# Terminal 2 — dispara o client
cd client
go run client.go
cat cotacao.txt
```

## Notas técnicas

- Driver SQLite usado: `modernc.org/sqlite`.