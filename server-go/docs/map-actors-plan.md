# Map actors — plano de implementação completa

Status: plano aprovado para execução  
Último código relevante: `2cdc604 perf(maps): run map simulation in per-map actors`  
Relatório de comparação: `docs/load-test-baseline.md`

## 1. Veredito do estado atual

O código atual é a **v1 híbrida**. Ela não atende ao desenho combinado:

| Área | Estado atual | Alvo |
|---|---|---|
| Tick de simulação | 1 goroutine por mapa (`runMapActor`) | 1 ator por mapa, dono de todo o runtime local |
| Comandos de jogador | Goroutine central (`World.loop`) | Mailbox do mapa dono do jogador |
| Player runtime | Compartilhado entre coordinator e mapa | Um único mapa dono por vez |
| Sincronização | `World.rw sync.RWMutex` global | Nenhum mutex sobre estado de mapa |
| Transferência | `transferPlayer` muta dois mapas no coordinator | Detach por snapshot + attach no destino |
| Roteamento | Ponteiro `*Player` | `playerID + MapKey + epoch` |
| Eventos/autoruns | Coordinator (`tickEvents`, `tickAutoruns`) | Ator do mapa |
| Morte/respawn | Fila para o coordinator | Local ao mapa |
| Social | Coordinator, mas lê/escreve campos do `Player` do mapa | Coordinator dono de `onlineEntry` |
| Persistência | Coordinator lê `p.Character` | Mapa tira snapshot e enfileira no worker |
| Vida do mapa | Nunca dorme nem descarrega | Dorme vazio, descarrega após carência |
| Instâncias | Só `MapID` | `MapKey{TemplateID, InstanceID}` |

Evidências no código:

- `internal/game/world.go`: `World.players`, `World.maps`, `World.byCharacter` e o `rw.Lock()` em volta de `fn()` no `loop()`.
- `internal/game/world.go`: `ensureMap` cria `mapState` e a goroutine, mas o mapa só faz tick; os comandos continuam no coordinator.
- `internal/game/events.go`: `transferPlayer` remove/adiciona o mesmo `*Player` em dois `mapState`.
- `internal/game/battle.go`: `queuePlayerDied` devolve a morte ao coordinator.
- `internal/game/npcs.go`: `queueEventTouch` devolve o toque ao coordinator.
- `internal/game/guild.go` e `party.go`: mutam `GuildName`, `PartyID`, `Character.GuildID` e o inventário de outros jogadores.
- `internal/game/bank.go`: `openBank` faz SELECT no goroutine do mapa.
- `internal/game/persist.go`: o worker existe, mas o snapshot é tirado pelo coordinator.

A v1 não é o alvo e não deve ser tratada como base conceitual. O alvo abaixo substitui
o `rw`, o `World.players` como fonte de runtime e o roteamento por ponteiro.

## 2. Objetivo

Cada mapa ativo é um **ator**:

- tem inbox próprio;
- é dono exclusivo de `mapState` e do runtime dos jogadores presentes;
- processa comandos, eventos, combate, NPCs e drops na própria goroutine;
- não expõe estado mutável para fora;
- conversa com o coordinator por mensagens;
- dorme quando vazio e é destruído após carência.

O coordinator é outro ator:

- dono do diretório online;
- dono de guilda, party, amigos, requests sociais, switches globais, MOTD;
- dono do ciclo de vida dos mapas/instâncias;
- dono da persistência (worker dedicado);
- nunca espera por um mapa.

A sessão de rede:

- não guarda `*Player`;
- guarda `playerID`;
- entrega intents ao `World`, que resolve rota no coordinator e despacha ao mapa atual;
- valida `epoch` para descartar comando atrasado de antes de um transfer.

## 3. Ownership

### 3.1 Coordinator (`World.loop`)

| Dado | Dono | Observação |
|---|---|---|
| `online map[uint16]*onlineEntry` | coordinator | diretório autoritativo |
| `byCharacter map[int64]uint16` | coordinator | unicidade online |
| `maps map[MapKey]*mapActor` | coordinator | ciclo de vida |
| `guilds`, `guildByName` | coordinator | social |
| `parties`, `nextPartyID` | coordinator | social |
| `globalSwitches`, `motd` | coordinator | globais |
| `nextID`, `nextEpoch` | coordinator | identidade/rota |
| `persistWorker` | coordinator | fila do banco |

`onlineEntry`:

```go
type onlineEntry struct {
    ID          uint16
    CharacterID int64
    Name        string        // imutável durante a sessão
    Account     *store.Account
    Client      Client
    IP          string
    Key         MapKey
    Epoch       uint64
    GuildName   string
    PartyID     int
    Friends     []string
    MuteUntil   time.Time
    CreatingGuild bool
    Request     *pendingRequest // apenas party/guild/friend
}
```

### 3.2 Mapa (`mapActor`)

Dono exclusivo de:

- `mapState` (`npcs`, `enemies`, `drops`, `autorun`);
- `Player` de cada jogador presente;
- `Character`, `CharacterState`, `Bank`, `Status`;
- posição, direção, HP/MP/EXP/gold, inventário, equips, skills, quests, switches/variables;
- `Event`, `Shop`, `Trade`, `Target`, timers de movimento/ataque/chat;
- cópia local de `guildName` e `partyID` usada em snapshot, drops e exp de party.

O mapa **não** lê nem escreve:

- `onlineEntry`, guildas, parties, amigos, requests sociais, `nextID`, `persistWorker`.

### 3.3 Sessão (`netserver.Session`)

- `account`, `characters`, `pending`, `authenticated` continuam na sessão;
- `player *game.Player` é substituído por `playerID uint16` e `inGame bool`;
- todos os `World.X(ctx, s.player, ...)` passam a `World.X(ctx, s.playerID, ...)`.

## 4. Primitivas

```go
type MapKey struct {
    TemplateID int
    InstanceID int // 0 = mundo aberto
}

type Route struct {
    Key      MapKey
    Epoch    uint64
    PlayerID uint16
}

type mapTask struct {
    playerID uint16
    epoch    uint64
    run      func(*mapActor, *Player) error
    reply    chan error
}

type mapActor struct {
    key   MapKey
    world *World
    ms    *mapState

    inbox chan mapTask
    ctrl  chan func(*mapActor)
    stop  chan struct{}
    done  chan struct{}

    idleSince time.Time
    players   []*Player // id-indexed, espelho de ms.players
}
```

Regras:

1. `ms` só é lido/escrito na goroutine do ator.
2. Nenhum `*mapState`, `*Player` ou `*npcState` sai do ator.
3. Mensagens carregam valores imutáveis (snapshot, IDs, pacotes prontos).
4. O coordinator nunca bloqueia esperando um mapa.
5. O mapa pode bloquear esperando o coordinator (request/response).
6. Toda transferência é `detach` (snapshot) seguido de `attach` (novo runtime).
7. Nenhum comando executa se `task.epoch != player.epoch`.
8. Fechar um mapa é ordenado: parar inbox de jogo, desanexar jogadores, encerrar.

Consequência: não existe caminho de deadlock coordinator↔mapa, porque só há espera
em uma direção.

## 5. Ciclo de vida

### 5.1 Ativação

`ensureMap(key)` no coordinator:

- valida template;
- cria `mapState` a partir do template imutável;
- cria `mapActor`;
- inicia `go actor.run()`;
- registra em `w.maps[key]`.

Mapas são criados sob demanda (primeiro attach) e não no boot.

### 5.2 Attach

Mensagem `attachPlayer` enviada pelo coordinator ao ator de destino:

```go
type attachPlayer struct {
    snap    *playerSnapshot
    epoch   uint64
    key     MapKey
    enter   bool // enviar EnterWorld
}
```

No ator:

1. valida posição;
2. materializa `*Player` a partir do snapshot;
3. aplica `epoch` e `key`;
4. insere em `ms.players` e no índice local;
5. envia `EnterWorld`, eventos, drops, jogadores e inimigos;
6. atualiza `idleSince`.

Falha de attach: envia `attachFailed` ao coordinator, que derruba a sessão com erro.

### 5.3 Detach

Mensagem `detachPlayer`:

1. remove do índice local;
2. limpa `trade`, `target`, `event` (se houver);
3. snapshot profundo de `Character`, `CharacterState`, `Bank`, `Status` e campos locais;
4. enfileira snapshot no `persistWorker`;
5. broadcast `PlayerLeft`;
6. responde o snapshot.

### 5.4 Transfer

Fluxo assíncrono, sem deadlock:

1. comando no mapa de origem (evento/teleport) valida destino com dados imutáveis;
2. origem faz detach e envia `transferCommit{snap, destKey, x, y, dir}` ao coordinator;
3. coordinator valida bounds, incrementa `epoch`, atualiza `onlineEntry` e envia
   `attachPlayer` ao destino;
4. destino materializa o player, envia `Transfer` e o restante do estado;
5. se o destino falhar, o coordinator devolve `attachPlayer` à origem com o snapshot.

O cliente não recebe `Transfer` antes do attach bem-sucedido.

### 5.5 Sleep e unload

- mapa sem jogadores para de tickar imediatamente (não processa `tick`);
- após `MapIdleSeconds` (padrão 60 s) envia `mapIdle` ao coordinator;
- coordinator remove do registry e fecha o ator;
- próximo attach recria o mapa.

Não há persistência de estado do mapa (decisão atual).

### 5.6 Stop

1. coordinator para de aceitar comandos;
2. envia `shutdown` a todos os atores;
3. cada ator tira snapshot de todos os players e enfileira no worker, depois encerra;
4. coordinator espera `done` de cada ator;
5. fecha e faz flush do `persistWorker`.

Durante `shutdown` o ator não chama o coordinator.

## 6. Comandos de jogador

`World` expõe a mesma API pública, mas a assinatura passa a usar `playerID`:

```go
func (w *World) Move(ctx context.Context, id uint16, direction uint8) error
```

Implementação:

```go
func (w *World) onPlayer(ctx context.Context, id uint16,
    fn func(*mapState, *Player) error) error {

    route, err := w.lookupRoute(ctx, id) // coordinator, leitura
    if err != nil { return err }
    task := mapTask{playerID: id, epoch: route.Epoch, run: fn}
    return w.sendMapTask(ctx, route.Key, task)
}
```

`sendMapTask`:

- `Inline` (testes): executa no coordinator, sem goroutine;
- produção: envia ao `inbox` do ator e espera `reply`, com `ctx` e `w.stop`.
- se `epoch` não bater: `ErrStaleRoute`; o chamador tenta uma vez com rota nova.

Comandos roteados ao mapa:

`Move`, `Attack`, `SetTarget`, `UseSkill`, `UseItem`, `UseHotbar`,
`Equip`, `SetHotbar`, `SpendPoint`, `EventNext`, `EventChoice`, `Teleport`,
`ShopBuy`, `ShopSell`, `BankItem`, `BankGold`, `DropItem`, `PickupDrop`,
`CloseWindow`, `TradeItem`, `TradeGold`, `RequestFinishTrade`, `AcceptRequest`,
`DeclineRequest`, `PlayerRequest`, `CreateGuild`.

Comandos no coordinator:

`Chat`, `OpenGuild`, `SetGuildLeader`, `SetGuildNotice`, `KickGuildMember`,
`InviteGuildMember`, `LeaveGuild`, `LeaveParty`, `OpenFriends`, `RemoveFriend`,
`CharacterOnline`, `DeleteCharacter`, `Submit`.

`AdminCommand` é roteado ao mapa do admin; ações sobre outros jogadores são
mensagens assíncronas do mapa do admin para o mapa do alvo. Ban/unban/MOTD/switch
global são pedidos síncronos do mapa ao coordinator.

## 7. Tick do mapa

Uma única goroutine por ator:

```go
for {
    select {
    case task := <-inbox:    apply(task)
    case <-ticker.C:         tick(actor)
    case <-stop:             return
    }
}
```

`tick` executa, nesta ordem:

1. `tickCombatMap(ms, now)`
2. `tickStatus`/respawn/recover (já dentro do combate)
3. `tickEventsMap(ms, now)` para eventos em espera de cada player
4. `tickAutorunsMap(ms, now)`
5. `tickNPCsMap(ms, now)`
6. `tickDropsMap(ms, now)`
7. persistência periódica dos players do mapa

Eventos que antes voltavam ao coordinator:

- `queueEventTouch` → chamada direta `eventTouch` no ator;
- `queuePlayerDied` → chamada direta `playerDied` no ator;
- respawn/clear status → local.

Party exp/gold:

- `partyMembersOnMap` passa a varrer `ms.players` comparando `p.partyID`;
- não há leitura de `Character` de outro mapa.

## 8. Social no coordinator

### 8.1 Guild

- estado, DB e chat no coordinator;
- `CreateGuild` é comando do mapa do jogador; o mapa chama `coordinator.createGuild(id, name, flag)`;
- coordinator atualiza `onlineEntry`, DB e envia `socialPatch{GuildID, GuildName}` ao mapa do jogador;
- kick/leave/delete enviam `socialPatch` aos mapas dos jogadores online afetados;
- `broadcastGuildName` vira envio direto do coordinator aos `Client` dos jogadores na mesma `MapKey` (packets não tocam estado de mapa);
- `Character.GuildID` só é mutado no mapa via patch.

### 8.2 Party

- roster e convites no coordinator;
- convite é comando do mapa do convidante, que valida distância/instância nos jogadores locais e chama `coordinator.inviteParty`;
- aceite é comando do mapa do aceitante, que chama `coordinator.acceptParty` e recebe as rotas dos membros;
- mensagem de entrada de party de cada membro é montada pelo mapa de origem do membro e enviada ao `Client` do novo membro por mensagem assíncrona mapa→mapa;
- exp/gold de party usam apenas membros presentes no mesmo `ms`;
- `PartyID` do runtime é atualizado por `socialPatch`.

### 8.3 Friends

- lista e convites no coordinator;
- `Friends` é copiado para o snapshot de attach;
- `OpenFriends`/`RemoveFriend` no coordinator;
- nenhum campo de friends é lido no tick do mapa.

### 8.4 Trade

- apenas mesma instância de mapa;
- convite, oferta, ouro e conclusão rodam inteiramente no ator do mapa;
- `Trade` e `pendingRequest` de trade são runtime do mapa;
- a troca de itens/ouro é local, sem coordinator e sem DB no caminho crítico;
- persistência dos dois jogadores é enfileirada no worker.

### 8.5 Chat

- todo chat passa pelo coordinator;
- canal mapa: coordinator itera `onlineEntry` com a mesma `MapKey` e envia `ServerMapChat`;
- canais global/party/guild/private: coordinator;
- mute em `onlineEntry`.

## 9. Persistência

- `persistWorker` continua único e fora dos mapas;
- o ator tira o snapshot e chama `worker.queue(character, state)`;
- `worker.queue` copia de novo (defensivo) e nunca bloqueia;
- save periódico: ticker do próprio ator;
- save de detach: imediato;
- save de ação (`saveQuietly`): enfileira, sem esperar DB;
- banco (`BankState`): carregado pelo coordinator no join e entregue no snapshot de attach;
- `add-vip-days`: script do mapa chama `coordinator.addVIPDays` (DB fora do mapa).

## 10. Instâncias

- `MapKey{TemplateID: id, InstanceID: 0}` para o mundo aberto;
- `Character.MapID` continua sendo o template;
- transfer futuro aceitará `InstanceID` no comando de evento;
- nenhuma instância é criada até existir conteúdo que a peça;
- a chave já entra em todo o roteamento para não refatorar depois.

## 11. Configuração

Novos campos server-only:

```toml
[server]
map_idle_seconds = 60
map_unload_seconds = 120
map_mailbox = 512
```

- `map_idle_seconds`: tempo sem jogadores até parar o tick;
- `map_unload_seconds`: tempo sem jogadores até destruir o ator;
- `map_mailbox`: capacidade do inbox.

Testes usam `Options.InlineMaps = true` e mantêm `Tick` alto.

## 12. Fases de implementação

### Fase 1 — Infraestrutura do ator

Arquivos: `internal/game/mapactor.go` (novo), `world.go`.

- `MapKey`, `Route`, `mapTask`, `mapActor`, `playerSnapshot`;
- registry `w.maps` por `MapKey` e diretório `w.online`;
- `ensureMap`, `attachPlayer`, `detachPlayer`, `lookupRoute`;
- `InlineMaps`;
- tick do ator (comandos ainda podem entrar pelo coordinator nesta fase).

Aceite: `go test ./...` verde; mapa ativado por attach; nenhum acesso a `ms`
fora do ator em produção.

### Fase 2 — Roteamento de comandos

Arquivos: `world.go`, `battle.go`, `character.go`, `bank.go`, `drops.go`,
`shop.go`, `events.go`, `trade.go`, `admin.go`.

- trocar `w.do` por `w.onPlayer` nos comandos de mapa;
- mudar `netserver` para `playerID`;
- remover `World.players` como fonte de runtime;
- `SetTarget`/PvP usam índice local do mapa.

Aceite: comandos concorrentes não tocam `ms`; `-race` verde com 50 bots.

### Fase 3 — Eventos, tick e transferência

Arquivos: `events.go`, `npcs.go`, `battle.go`, `world.go`.

- `tickEventsMap`, `tickAutorunsMap`, `tickCombatMap`, `tickNPCsMap`, `tickDropsMap`;
- remover `queueEventTouch` e `queuePlayerDied`;
- detach/attach por snapshot no transfer;
- `epoch` em cada task.

Aceite: testes de evento/transfer/npc/drop verdes; sem `rw`.

### Fase 4 — Social

Arquivos: `guild.go`, `party.go`, `friends.go`, `trade.go`, `admin.go`, `world.go`.

- `onlineEntry` como fonte social;
- patches `socialPatch` para o mapa;
- guild/party/friends no coordinator;
- trade local ao mapa;
- chat no coordinator.

Aceite: testes de guild/party/friends/trade verdes; mapas não leem estado social.

### Fase 5 — Persistência e banco

Arquivos: `persist.go`, `bank.go`, `character.go`, `eventscript.go`.

- snapshot de detach e periódico no ator;
- bank pré-carregado no join;
- VIP via coordinator;
- `World.Stop` encerra atores e faz flush.

Aceite: testes de persistência/banco verdes; nenhum SELECT/UPDATE no ator.

### Fase 6 — Sleep/unload e observabilidade

Arquivos: `world.go`, `mapactor.go`, `config`.

- idle/unload;
- logs `map activated`, `map asleep`, `map unloaded`, `map attach failed`;
- contadores de inbox/rejeições.

Aceite: mapa vazio para de tickar e é destruído; próximo attach recria.

### Fase 7 — Validação final

- `go test ./...`, `go vet ./...`;
- 50 bots com `-race`;
- smoke com cliente real;
- 500 bots com o workload do baseline;
- `docs/load-test-post-parallel.md` comparando com `docs/load-test-baseline.md`.

## 13. Testes

- manter `InlineMaps` para os testes existentes;
- adicionar testes de ator real (sem inline):
  - dois mapas em goroutines distintas com 50 players cada;
  - transferência concorrente com comandos atrasados (epoch);
  - attach/detach sem vazamento de ponteiro;
  - mapa vazio dorme/descarrega;
  - trade entre jogadores do mesmo mapa e recusa entre mapas;
  - guild/party patches chegam ao runtime;
  - save de detach no worker.

## 14. Riscos e decisões

- **Snapshot profundo**: obrigatório para `Character`, `CharacterState`,
  `Bank`, `Status`, `Trade`, `ErroredEvents`, `ShopGoods`.
- **Evento em transferência**: transferência encerra o evento atual (comportamento
  já existente via `return true`); o snapshot leva `Event = nil`.
- **Attach assíncrono**: comandos entre detach e attach podem receber
  `ErrNotInGame`; o cliente reenvia no próximo tick. Não há perda de estado.
- **Mapa com 1 jogador**: broadcast O(1); com N jogadores continua O(N²) no mapa,
  mas agora isolado por ator.
- **Sem TLS**: fora do escopo.
- **Sem persistência de mapa**: decisão mantida.
- **Sem reactor de rede por mapa**: mantido; o netpoll do Go é global.

## 15. Ordem de commits

1. `docs(maps): write the complete map-actor migration plan`
2. `refactor(maps): add actor runtime, registry and online directory`
3. `refactor(maps): route player intents through map actors`
4. `refactor(maps): own events, ticks and player transfer in the actor`
5. `refactor(social): move social state to coordinator entries`
6. `refactor(persist): snapshot from map actors and preload banks`
7. `perf(maps): sleep and unload idle maps`
8. `test(maps): actor, transfer and race coverage`
9. `test(load): post-parallel 500-bot report`

O trabalho só é considerado concluído quando as fases 1–7 estiverem no mesmo
estado de `main`, sem `rw`, sem `World.players` como runtime e com o relatório
final comparando o baseline.
