# Git Dashboard — Documento de Projeto e Guia de Estudo

> **Papel deste documento:** especificação técnica inicial e mapa de estudos.  
> **Projeto:** aplicação TUI para inspecionar repositórios Git, escrita em Go e construída com o ecossistema Charm.  
> **Postura de desenvolvimento:** implementação manual, consulta à documentação oficial e decisões técnicas justificadas pelo desenvolvedor.

---

## 0. Como usar este documento

Este documento descreve **o que o produto deve oferecer**, quais problemas precisam ser tratados e quais APIs merecem estudo. Ele não é um tutorial passo a passo nem uma implementação de referência.

Use as seções como mapa de navegação:

1. Leia a visão do produto e os limites do primeiro release.
2. Consulte os conceitos de Go necessários para entender filesystem, processos externos, erros e concorrência.
3. Estude a documentação do Git para saber quais informações podem ser solicitadas ao executável `git`.
4. Estude Bubble Tea e, depois, os componentes Bubbles e os recursos visuais de Lip Gloss.
5. Consulte os critérios de aceite e os cenários de teste durante o desenvolvimento.

Os exemplos de código deste documento são **microexemplos de APIs isoladas**. Eles demonstram o formato de entrada, o tipo de retorno ou um comportamento de biblioteca. Não formam uma solução pronta para o dashboard.

### Regra de pesquisa

A documentação oficial de `pkg.go.dev` é a referência para assinaturas e contratos de pacotes Go. O README, os exemplos e as notas de migração de cada repositório Charm ajudam a entender a intenção de uso. Quando houver divergência entre um tutorial antigo e a documentação da versão instalada, prevalece a documentação da versão instalada.

---

### Go — linguagem, ferramentas e biblioteca padrão

| Assunto | Documentação |
|---|---|
| Tour interativo da linguagem | [A Tour of Go](https://go.dev/tour/) |
| Documentação geral da linguagem | [go.dev/doc](https://go.dev/doc/) |
| Especificação da linguagem | [Go Language Specification](https://go.dev/ref/spec) |
| Effective Go | [Effective Go](https://go.dev/doc/effective_go) |
| Módulos e dependências | [Go Modules Reference](https://go.dev/ref/mod) |
| Comandos `go` | [Go Command Documentation](https://go.dev/doc/cmd) |
| Testes | [Tutorial: Add a test](https://go.dev/doc/tutorial/add-a-test) |
| Race detector | [Data Race Detector](https://go.dev/doc/articles/race_detector) |
| Concorrência | [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines) |
| Contexto e cancelamento | [Go blog: Context](https://go.dev/blog/context) |

#### Pacotes da biblioteca padrão usados ou relevantes

| Pacote | Referência `pkg.go.dev` | O que consultar |
|---|---|---|
| `os` | [pkg.go.dev/os](https://pkg.go.dev/os) | `Getwd`, `Stat`, `ReadDir`, `Open`, `ReadFile`, `LookupEnv`, `Getenv`, `Chdir` |
| `path/filepath` | [pkg.go.dev/path/filepath](https://pkg.go.dev/path/filepath) | `Abs`, `Join`, `Clean`, `Rel`, `IsAbs`, `WalkDir`, `Glob` |
| `os/exec` | [pkg.go.dev/os/exec](https://pkg.go.dev/os/exec) | `Command`, `CommandContext`, `LookPath`, `Cmd.Output`, `Cmd.CombinedOutput`, `Cmd.Run`, `Cmd.Start`, `Cmd.Wait`, `ExitError` |
| `context` | [pkg.go.dev/context](https://pkg.go.dev/context) | `Background`, `TODO`, `WithCancel`, `WithTimeout`, `WithDeadline`, `CancelFunc` |
| `os/signal` | [pkg.go.dev/os/signal](https://pkg.go.dev/os/signal) | `NotifyContext`, `Stop` |
| `strings` | [pkg.go.dev/strings](https://pkg.go.dev/strings) | `TrimSpace`, `Split`, `Cut`, `Fields`, `Builder`, `Contains` |
| `bufio` | [pkg.go.dev/bufio](https://pkg.go.dev/bufio) | `NewScanner`, `Scanner.Scan`, `Scanner.Text`, `Scanner.Bytes`, `Scanner.Err`, `Reader` |
| `bytes` | [pkg.go.dev/bytes](https://pkg.go.dev/bytes) | `Buffer`, `Reader`, `Split`, `TrimSpace` |
| `errors` | [pkg.go.dev/errors](https://pkg.go.dev/errors) | `New`, `Is`, `As`, `Join`, wrapping de erros |
| `fmt` | [pkg.go.dev/fmt](https://pkg.go.dev/fmt) | `Errorf`, `Fprintf`, `Sprintf`, `Sprint` |
| `io` | [pkg.go.dev/io](https://pkg.go.dev/io) | `Reader`, `Writer`, `ReadAll`, `Copy`, `EOF` |
| `encoding/json` | [pkg.go.dev/encoding/json](https://pkg.go.dev/encoding/json) | `Marshal`, `Unmarshal`, `NewDecoder`, `Decoder.Decode` |
| `testing` | [pkg.go.dev/testing](https://pkg.go.dev/testing) | `T`, `TempDir`, `Run`, `Helper`, `Cleanup`, `F` |
| `runtime` | [pkg.go.dev/runtime](https://pkg.go.dev/runtime) | `GOOS`, `GOARCH`, `NumCPU` — útil para entender diferenças de plataforma |

Links diretos de símbolos úteis:

- [`os.Getwd`](https://pkg.go.dev/os#Getwd) · [`os.Stat`](https://pkg.go.dev/os#Stat) · [`os.ReadDir`](https://pkg.go.dev/os#ReadDir) · [`os.Open`](https://pkg.go.dev/os#Open) · [`os.ReadFile`](https://pkg.go.dev/os#ReadFile)
- [`filepath.Abs`](https://pkg.go.dev/path/filepath#Abs) · [`filepath.Join`](https://pkg.go.dev/path/filepath#Join) · [`filepath.WalkDir`](https://pkg.go.dev/path/filepath#WalkDir)
- [`exec.Command`](https://pkg.go.dev/os/exec#Command) · [`exec.CommandContext`](https://pkg.go.dev/os/exec#CommandContext) · [`exec.Cmd.Output`](https://pkg.go.dev/os/exec#Cmd.Output) · [`exec.Cmd.CombinedOutput`](https://pkg.go.dev/os/exec#Cmd.CombinedOutput) · [`exec.Cmd.Run`](https://pkg.go.dev/os/exec#Cmd.Run) · [`exec.LookPath`](https://pkg.go.dev/os/exec#LookPath)
- [`context.WithCancel`](https://pkg.go.dev/context#WithCancel) · [`context.WithTimeout`](https://pkg.go.dev/context#WithTimeout) · [`signal.NotifyContext`](https://pkg.go.dev/os/signal#NotifyContext)
- [`strings.TrimSpace`](https://pkg.go.dev/strings#TrimSpace) · [`strings.Cut`](https://pkg.go.dev/strings#Cut) · [`bufio.NewScanner`](https://pkg.go.dev/bufio#NewScanner)
- [`errors.Is`](https://pkg.go.dev/errors#Is) · [`errors.As`](https://pkg.go.dev/errors#As) · [`testing.T.TempDir`](https://pkg.go.dev/testing#T.TempDir)

### Charmbracelet — documentação e pacotes

| Projeto | API / documentação | Repositório / material de estudo |
|---|---|---|
| Bubble Tea v2 | [pkg.go.dev/charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) | [GitHub: bubbletea](https://github.com/charmbracelet/bubbletea) · [Tutorials](https://github.com/charmbracelet/bubbletea/tree/main/tutorials) · [Examples](https://github.com/charmbracelet/bubbletea/tree/main/examples) |
| Bubbles v2 | [pkg.go.dev/charm.land/bubbles/v2](https://pkg.go.dev/charm.land/bubbles/v2) | [GitHub: bubbles](https://github.com/charmbracelet/bubbles) · [Upgrade guide v2](https://github.com/charmbracelet/bubbles/blob/main/UPGRADE_GUIDE_V2.md) |
| Bubbles — list | [pkg.go.dev/charm.land/bubbles/v2/list](https://pkg.go.dev/charm.land/bubbles/v2/list) | Lista navegável |
| Bubbles — table | [pkg.go.dev/charm.land/bubbles/v2/table](https://pkg.go.dev/charm.land/bubbles/v2/table) | Tabelas e seleção de linhas |
| Bubbles — viewport | [pkg.go.dev/charm.land/bubbles/v2/viewport](https://pkg.go.dev/charm.land/bubbles/v2/viewport) | Conteúdo longo com rolagem; útil para diff |
| Bubbles — textinput | [pkg.go.dev/charm.land/bubbles/v2/textinput](https://pkg.go.dev/charm.land/bubbles/v2/textinput) | Campo de texto |
| Bubbles — textarea | [pkg.go.dev/charm.land/bubbles/v2/textarea](https://pkg.go.dev/charm.land/bubbles/v2/textarea) | Texto multilinha |
| Bubbles — spinner | [pkg.go.dev/charm.land/bubbles/v2/spinner](https://pkg.go.dev/charm.land/bubbles/v2/spinner) | Indicador de carregamento |
| Bubbles — help | [pkg.go.dev/charm.land/bubbles/v2/help](https://pkg.go.dev/charm.land/bubbles/v2/help) | Ajuda de atalhos |
| Bubbles — key | [pkg.go.dev/charm.land/bubbles/v2/key](https://pkg.go.dev/charm.land/bubbles/v2/key) | Mapeamento de teclas |
| Lip Gloss v2 | [pkg.go.dev/charm.land/lipgloss/v2](https://pkg.go.dev/charm.land/lipgloss/v2) | [GitHub: lipgloss](https://github.com/charmbracelet/lipgloss) · [Examples](https://github.com/charmbracelet/lipgloss/tree/main/examples) |
| Huh v2 | [pkg.go.dev/charm.land/huh/v2](https://pkg.go.dev/charm.land/huh/v2) | [GitHub: huh](https://github.com/charmbracelet/huh) · Formulários e prompts |
| Glamour v2 | [pkg.go.dev/charm.land/glamour/v2](https://pkg.go.dev/charm.land/glamour/v2) | [GitHub: glamour](https://github.com/charmbracelet/glamour) · Renderização Markdown |
| Log v2 | [pkg.go.dev/charm.land/log/v2](https://pkg.go.dev/charm.land/log/v2) | [GitHub: log](https://github.com/charmbracelet/log) · Logging |
| Harmonica | [pkg.go.dev/github.com/charmbracelet/harmonica](https://pkg.go.dev/github.com/charmbracelet/harmonica) | [GitHub: harmonica](https://github.com/charmbracelet/harmonica) · Animações por mola |
| Gum | [GitHub: gum](https://github.com/charmbracelet/gum) | Utilitários interativos para shell; não é dependência central da TUI |
| Fang | [GitHub: fang](https://github.com/charmbracelet/fang) | Integração de CLI com Cobra; opcional para uma futura CLI com subcomandos |
| Wish | [pkg.go.dev/charm.land/wish/v2](https://pkg.go.dev/charm.land/wish/v2) | [GitHub: wish](https://github.com/charmbracelet/wish) · Servir TUIs por SSH |

#### Documentação adicional da Charm

- [Organização Charmbracelet no GitHub](https://github.com/charmbracelet)
- [Site oficial Charm](https://charm.sh/)
- [Bubble Tea — documentação do projeto](https://github.com/charmbracelet/bubbletea)
- [Bubbles — catálogo de componentes](https://github.com/charmbracelet/bubbles)
- [Lip Gloss — documentação e exemplos](https://github.com/charmbracelet/lipgloss)

### Git — comandos e formatos consultados pelo projeto

- [`git status`](https://git-scm.com/docs/git-status) — status legível e formato `--porcelain` / `-z`.
- [`git rev-parse`](https://git-scm.com/docs/git-rev-parse) — raiz do repositório, `HEAD` e referências.
- [`git branch`](https://git-scm.com/docs/git-branch) — branch atual e listagem de branches.
- [`git diff`](https://git-scm.com/docs/git-diff) — diferenças do working tree e do index.
- [`git log`](https://git-scm.com/docs/git-log) — histórico de commits.
- [`git show`](https://git-scm.com/docs/git-show) — detalhes de objetos e commits.
- [Git Reference](https://git-scm.com/docs) — índice completo de comandos.

### Como estudar cada referência

1. Abra primeiro a página do pacote, não apenas a página de uma função.
2. Leia o resumo do pacote e os tipos exportados.
3. Abra as funções e métodos citados na tabela; observe os tipos de retorno e os erros possíveis.
4. Leia os exemplos oficiais e, quando disponível, abra o código-fonte relacionado.
5. Confira a versão exibida no topo do `pkg.go.dev` e compare com o `go.mod` do projeto.
6. Para os módulos Charm v2 que compartilham tipos, mantenha Bubble Tea, Bubbles, Lip Gloss e Huh na mesma geração principal. Não misture exemplos v1 com imports v2 sem entender as mudanças.

---

# 1. Visão do produto

## 1.1 Objetivo

Criar uma aplicação de terminal que apresente, em uma única interface, um resumo legível do repositório Git selecionado.

A aplicação deve reduzir a necessidade de executar vários comandos Git manualmente apenas para responder perguntas como:

- Em qual repositório estou?
- Qual é a branch atual?
- Qual é o commit mais recente?
- Existem alterações locais?
- Quais arquivos foram modificados, adicionados, removidos ou estão sem rastreamento?
- Quais alterações estão no working tree e quais já estão no index (staging area)?
- Qual é o conteúdo do diff de um arquivo selecionado?

O produto deve ser útil mesmo sem executar operações que alterem o repositório.

## 1.2 Público e ambiente inicial

- Aplicação local, executada em um terminal interativo.
- Sistema inicial de desenvolvimento: Linux.
- Linguagem: Go.
- Fonte das informações Git: executável `git` instalado no sistema.
- Interface: Bubble Tea.
- Componentes de interface: Bubbles, quando fizer sentido.
- Estilo e composição visual: Lip Gloss.

A compatibilidade com macOS e Windows pode ser considerada desde o início nas decisões de filesystem e execução de processos, mas não precisa bloquear o primeiro release funcional.

## 1.3 Princípio arquitetural

Separar três responsabilidades:

1. **Coleta:** localizar o repositório e consultar o Git.
2. **Aplicação:** representar os dados, controlar o estado da interface e decidir o que acontece quando o usuário interage.
3. **Apresentação:** renderizar os dados e os controles no terminal.

A interface não deve conhecer detalhes de parsing da saída do Git. A camada de coleta não deve depender do Bubble Tea ou de Lip Gloss. Os dados do repositório devem poder ser testados sem iniciar uma TUI.

---

# 2. Escopo funcional

## 2.1 Primeiro release — MVP somente de leitura

### REQ-01 — Descoberta do diretório

A aplicação deve conseguir identificar o diretório de trabalho atual e aceitar um diretório-alvo como conceito de entrada.

Comportamentos esperados:

- Resolver caminhos relativos e absolutos.
- Lidar com espaços e caracteres Unicode no caminho.
- Identificar quando o caminho não existe ou não pode ser acessado.
- Distinguir o diretório informado do diretório raiz real do repositório.
- Se iniciada dentro de uma subpasta do repositório, reconhecer o repositório pai.

### REQ-02 — Detecção de repositório

A aplicação deve detectar se o diretório pertence a uma working tree Git.

Casos que precisam ser considerados:

- Diretório que não é um repositório.
- Repositório normal.
- Repositório iniciado, mas ainda sem commit inicial (unborn branch).
- Estado de `detached HEAD`.
- Subdiretório de um repositório.
- Git não instalado ou não encontrado no `PATH`.
- Repositório inacessível ou com metadados inválidos.

Não presuma que a existência de uma pasta `.git` é a única forma válida de identificar um repositório. Worktrees e submódulos podem usar um arquivo `.git` que aponta para metadados localizados em outro lugar. Prefira perguntar ao próprio Git.

### REQ-03 — Informações gerais

Exibir, no mínimo:

- Nome ou caminho da raiz do repositório.
- Branch atual, quando existir.
- Indicador claro para `detached HEAD`.
- Hash abreviado do commit atual, quando existir.
- Assunto (subject) do commit mais recente.
- Autor e data do último commit, caso os dados estejam disponíveis.

### REQ-04 — Status do working tree

Exibir um resumo de:

- Arquivos modificados.
- Arquivos adicionados.
- Arquivos removidos.
- Arquivos não rastreados (untracked).
- Alterações staged.
- Alterações unstaged.
- Conflitos, caso existam.

O estado limpo precisa ser representado explicitamente. Uma lista vazia de arquivos não deve ser interpretada como falha de leitura.

### REQ-05 — Lista de arquivos alterados

O usuário deve conseguir navegar pelos arquivos identificados no status do Git.

Para cada arquivo, a aplicação deve manter informação suficiente para distinguir:

- Estado no index (staging area).
- Estado no working tree.
- Arquivo novo ou sem rastreamento.
- Remoção.
- Renomeação, quando detectável.
- Conflito, quando detectável.

Não reduza o status do Git a um único rótulo simplificado antes de preservar os dados de origem. O formato curto do Git tem duas posições de status, que representam áreas diferentes.

### REQ-06 — Visualização de diff

Ao selecionar um arquivo compatível, o dashboard deve poder exibir o diff correspondente em uma área navegável.

O MVP deve diferenciar, conceitualmente:

- Diferenças entre working tree e index.
- Diferenças entre index e `HEAD`.

Arquivos não rastreados não possuem diff convencional contra `HEAD`; precisam de tratamento visual próprio ou de uma explicação adequada.

### REQ-07 — Atualização

O usuário deve conseguir solicitar uma nova leitura do repositório. A atualização não deve congelar a interface enquanto o Git é executado.

### REQ-08 — Feedback de estado

A interface deve comunicar:

- Carregamento.
- Atualização concluída.
- Erros recuperáveis.
- Diretório que não é repositório.
- Git ausente.
- Repositório sem commit inicial.
- Repositório em `detached HEAD`.

Mensagens de erro devem ajudar a entender o problema sem exigir que o usuário leia um stack trace.

## 2.2 Fora do MVP

Não implementar inicialmente:

- `git add` / stage.
- `git restore` ou descarte de alterações.
- `git commit`.
- `git checkout` ou `git switch`.
- `git merge` ou `git rebase`.
- `git push` / `git pull` / `git fetch`.
- Gerenciamento de remotes.
- Resolução interativa de conflitos.
- Autenticação com provedores Git.

Esses recursos podem entrar em versões posteriores, mas exigem confirmação de ações, tratamento de falhas e proteção contra perda de trabalho. A primeira versão deve ser uma ferramenta de inspeção segura.

---

# 3. Experiência de uso esperada

## 3.1 Layout conceitual

A interface pode ser dividida em áreas, sem impor uma composição visual específica:

| Área | Conteúdo possível |
|---|---|
| Cabeçalho | Nome/caminho do repositório, branch e hash atual |
| Resumo | Contagem de alterações por categoria e estado geral |
| Lista | Arquivos alterados, com status e seleção |
| Detalhes | Diff do arquivo selecionado ou informações do commit |
| Rodapé | Atalhos disponíveis, estado de carregamento e mensagens |

A composição final é uma decisão de produto. Não é necessário exibir todas as áreas simultaneamente em terminais pequenos.

## 3.2 Interações sugeridas

Estas são sugestões de comportamento, não uma exigência de teclas específicas:

| Ação | Resultado esperado |
|---|---|
| Sair | Encerrar a aplicação e restaurar o terminal |
| Atualizar | Solicitar nova leitura do repositório |
| Mover seleção | Navegar pela lista de arquivos |
| Abrir item | Exibir detalhes ou diff do arquivo selecionado |
| Alternar painel | Mudar entre resumo, lista e detalhes |
| Ajuda | Exibir os atalhos disponíveis |
| Redimensionar terminal | Recalcular dimensões sem quebrar o layout |

As teclas escolhidas devem ser consistentes, visíveis na ajuda e não devem conflitar com os controles dos componentes Bubbles.

## 3.3 Terminais pequenos

A aplicação deve continuar utilizável com largura e altura reduzidas:

- Evitar larguras fixas sem necessidade.
- Cortar ou quebrar caminhos longos de forma controlada.
- Permitir rolagem de diffs extensos.
- Não assumir que emojis ou caracteres Unicode ocupam uma única célula.
- Recalcular layout após receber eventos de redimensionamento.

---

# 4. Modelo mental de Bubble Tea

Bubble Tea implementa uma arquitetura inspirada na The Elm Architecture. O conceito central não é desenhar a tela manualmente a cada tecla; é manter um estado e produzir uma representação visual desse estado.

## 4.1 Os três elementos principais

### Model

Representa o estado atual da aplicação. No contexto deste projeto, o estado pode conter informações como diretório selecionado, snapshot do repositório, arquivo selecionado, painel ativo, tamanho do terminal, estado de carregamento e erro atual.

O `Model` não deve ser confundido com os dados do domínio Git. O modelo da TUI também guarda estado de navegação e apresentação.

### Msg (Message)

Representa um evento recebido pela aplicação. Pode ser uma tecla, redimensionamento de terminal, resultado de uma consulta Git, atualização de timer ou outro evento.

Uma mensagem de resultado de consulta é a ponte entre uma operação de I/O e a atualização do estado da interface.

### Cmd (Command)

Representa uma operação de I/O que o Bubble Tea executa fora do fluxo síncrono de renderização e que, ao terminar, produz uma mensagem.

Exemplos conceituais:

- Consultar branch e status.
- Ler um diff.
- Atualizar periodicamente a tela.
- Ler informações externas.

Uma regra importante: `View` deve ser uma função de apresentação. Não deve executar `git`, ler arquivos ou fazer I/O.

## 4.2 Ciclo de vida conceitual

1. O programa recebe um Model inicial.
2. `Init` pode solicitar uma primeira operação.
3. O Bubble Tea recebe mensagens de teclado, terminal e comandos concluídos.
4. `Update` interpreta a mensagem e produz um novo estado e, opcionalmente, novas operações.
5. `View` renderiza a interface com base no estado atual.
6. O ciclo continua até uma mensagem de encerramento.

O resultado de uma consulta Git deve voltar como mensagem tipada. Evite compartilhar estado mutável entre goroutines sem uma estratégia clara.

## 4.3 Assinaturas fundamentais — Bubble Tea v2

A interface atual do Bubble Tea v2 usa `tea.View` no método `View`. Muitos tutoriais antigos da versão v1 usam `View() string`; não misture os contratos.

```go
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() View
}
```

```go
type Cmd func() Msg
```

Leitura da assinatura:

- `Init() Cmd`: fornece uma operação inicial opcional.
- `Update(Msg) (Model, Cmd)`: recebe uma mensagem, devolve o estado atualizado e uma operação opcional.
- `View() View`: fornece a representação visual atual.
- `Cmd`: função executável que retorna uma `Msg` quando termina.

## 4.4 Atenção à versão 2

Use a documentação e os exemplos v2. Mudanças de versão incluem alterações de import path e APIs. A família atual de import paths documentada pelo Charm segue o padrão `charm.land/<projeto>/v2` para os projetos usados aqui.

Referências:

- [Bubble Tea v2 — pkg.go.dev](https://pkg.go.dev/charm.land/bubbletea/v2)
- [Bubble Tea — repositório e tutorial](https://github.com/charmbracelet/bubbletea)
- [Bubbles v2 — notas de lançamento/migração](https://github.com/charmbracelet/bubbles/releases)

---

# 5. Integração com Git

## 5.1 Estratégia recomendada

O projeto deve consultar o executável oficial `git` usando a biblioteca padrão `os/exec`.

Isso permite aprender:

- Como iniciar processos externos.
- Como passar argumentos separados.
- Como definir o diretório de execução.
- Como capturar `stdout` e `stderr`.
- Como interpretar exit codes.
- Como cancelar processos.
- Como transformar saída textual em tipos próprios do domínio.

Não é necessário começar com uma biblioteca Git escrita em Go. O próprio Git já conhece os formatos, regras e metadados do repositório instalado.

### Por que não executar uma string de shell?

A API `exec.Command` recebe o nome do programa e os argumentos separadamente. Ela não interpreta automaticamente operadores como `&&`, pipes, redirecionamentos, globbing ou variáveis de shell.

Prefira o modelo conceitual:

- Executável: `git`.
- Argumentos: `-C`, caminho, subcomando e opções.

Não concatene caminho ou entrada do usuário em uma string para passar a `sh -c`. Além de ser desnecessário neste caso, isso cria uma superfície de injeção de comandos.

## 5.2 Comandos Git de inspeção

Os comandos abaixo são referências de consulta para o desenvolvedor. São comandos de leitura; a documentação oficial de cada um contém todas as opções e exceções.

| Informação | Comando de referência | Observações sobre a saída |
|---|---|---|
| Raiz da working tree | `git -C <dir> rev-parse --show-toplevel` | Caminho da raiz; falha se o local não pertencer a uma working tree |
| Confirmar working tree | `git -C <dir> rev-parse --is-inside-work-tree` | Retorna `true` quando está dentro de uma working tree |
| Branch atual | `git -C <dir> branch --show-current` | Pode retornar saída vazia em `detached HEAD` |
| Commit atual | `git -C <dir> rev-parse --short HEAD` | Hash abreviado; pode falhar em repositório sem commit inicial |
| Estado resumido | `git -C <dir> status --porcelain=v1 --branch -z` | Formato estável para parsing por ferramenta |
| Diff unstaged | `git -C <dir> diff --name-status` | Mudanças entre working tree e index |
| Diff staged | `git -C <dir> diff --cached --name-status` | Mudanças entre index e `HEAD` |
| Estatística unstaged | `git -C <dir> diff --stat` | Resumo visual de linhas alteradas |
| Estatística staged | `git -C <dir> diff --cached --stat` | Resumo visual do que está no index |
| Último commit | `git -C <dir> log -1 --oneline` | Formato voltado a humanos; útil para exibição simples |
| Histórico curto | `git -C <dir> log -n <quantidade> --oneline --decorate` | Lista limitada de commits recentes |
| Branches locais | `git -C <dir> branch --list` | Listagem voltada a humanos; use opções `--format` se precisar de parsing mais previsível |

Para obter o diff completo de uma área específica, consulte a documentação de `git diff`; não confunda `git diff` com `git diff --cached`.

## 5.3 O formato de status do Git

A documentação de `git status` descreve um formato curto com duas posições, frequentemente representadas como `XY`:

- `X`: estado do index em relação ao commit `HEAD`.
- `Y`: estado do working tree em relação ao index.

Alguns exemplos conceituais:

| Código | Significado comum |
|---|---|
| ` M` | Arquivo modificado no working tree, ainda não staged |
| `M ` | Arquivo modificado no index (staged) |
| `A ` | Arquivo adicionado ao index |
| ` D` | Arquivo removido no working tree |
| `??` | Arquivo não rastreado |
| `UU` | Conflito de merge não resolvido em ambos os lados |

A tabela é ilustrativa, não uma gramática completa. Há combinações adicionais para renomeações, cópias, conflitos e outros estados.

### Regra de parsing

Não faça parsing do formato humano de `git status` dividindo a linha por espaços. Nomes de arquivos podem conter espaços, tabulações, aspas e outros caracteres especiais.

Para processamento por software:

- Prefira `--porcelain`.
- Estude a opção `-z`, que usa bytes NUL como separador de caminhos.
- Leia a documentação sobre registros de renomeação/cópia, que podem envolver mais de um caminho.
- Preserve os bytes/texto do caminho de acordo com a estratégia de compatibilidade escolhida.
- Não assuma que um nome de arquivo é uma única palavra.

O formato porcelain existe justamente para fornecer uma saída mais estável para scripts e ferramentas.

## 5.4 Situações Git que exigem tratamento

### Detached HEAD

`git branch --show-current` não imprime nome de branch quando `HEAD` está destacado. Isso não significa necessariamente que a consulta falhou. O dashboard deve reconhecer que existe um commit atual, mas não uma branch selecionada.

### Repositório sem commit inicial

Um repositório recém-inicializado pode não ter referência `HEAD` resolvida para um commit. Consultas ao hash e ao histórico podem falhar. O estado deve ser apresentado como “sem commit inicial”, não como um erro genérico inexplicável.

### Subdiretório

A aplicação pode ser iniciada dentro de `src/`, `cmd/` ou outra pasta interna. O Git deve ser consultado no contexto do diretório informado, mas os dados gerais devem identificar a raiz do repositório.

### Worktrees e submódulos

Não trate o diretório `.git` como necessariamente uma pasta comum. Consulte `git rev-parse` para resolver a localização e o contexto do repositório.

### Arquivos incomuns

Teste nomes com espaços, acentos, Unicode, colchetes e caracteres que normalmente são interpretados pelo shell. O argumento do processo deve permanecer um único argumento, e a saída do Git precisa ser interpretada segundo o formato escolhido.

## 5.5 Referências oficiais de Git

- [git-status](https://git-scm.com/docs/git-status)
- [git-rev-parse](https://git-scm.com/docs/git-rev-parse)
- [git-branch](https://git-scm.com/docs/git-branch)
- [git-diff](https://git-scm.com/docs/git-diff)
- [git-log](https://git-scm.com/docs/git-log)
- [git-show](https://git-scm.com/docs/git-show)

Prioridade de leitura: `git-status`, `git-rev-parse` e `git-diff` primeiro. Os demais podem ser consultados conforme a funcionalidade evoluir.

---

# 6. Biblioteca padrão Go — mapa de APIs

Esta seção funciona como um índice de estudo no estilo `pkg.go.dev`. As assinaturas são representativas das APIs relevantes; consulte a página oficial para documentação completa, comportamento por sistema operacional e detalhes de erro.

## 6.1 `os` — sistema operacional e filesystem

Documentação: [pkg.go.dev/os](https://pkg.go.dev/os)

### `os.Getwd`

**Assinatura:**

```go
func Getwd() (dir string, err error)
```

**O que faz:** retorna o diretório de trabalho atual do processo.

**Entrada:** nenhuma.

**Retorno:** caminho do diretório atual e erro, caso não seja possível obtê-lo.

**Exemplo isolado:**

```go
currentDirectory, err := os.Getwd()
```

**Uso conceitual no projeto:** descobrir em qual pasta o usuário executou o dashboard. Isso é equivalente ao propósito de consultar `$PWD`, mas não exige iniciar o comando externo `pwd`.

### `os.Stat`

**Assinatura:**

```go
func Stat(name string) (FileInfo, error)
```

**O que faz:** obtém informações sobre um caminho.

**Entrada:** caminho de arquivo ou diretório.

**Retorno:** um `FileInfo` com metadados ou um erro.

**Exemplo isolado:**

```go
info, err := os.Stat(path)
```

**Pontos de estudo:** `FileInfo.IsDir()`, `FileInfo.Name()`, `FileInfo.Size()` e diferenças entre arquivo inexistente e caminho inacessível.

### `os.ReadDir`

**Assinatura:**

```go
func ReadDir(name string) ([]DirEntry, error)
```

**O que faz:** lê as entradas de um diretório.

**Entrada:** caminho do diretório.

**Retorno:** slice de `DirEntry` e erro. As entradas retornadas são ordenadas por nome.

**Exemplo isolado:**

```go
entries, err := os.ReadDir(directory)
```

**Pontos de estudo:** `DirEntry.Name()`, `DirEntry.IsDir()`, `DirEntry.Type()` e `DirEntry.Info()`.

**Atenção:** `ReadDir` lista entradas de um diretório; não é uma API para encontrar automaticamente a raiz de um repositório Git.

### `os.LookupEnv`

**Assinatura:**

```go
func LookupEnv(key string) (value string, ok bool)
```

**O que faz:** consulta uma variável de ambiente e informa se ela existe.

**Entrada:** nome da variável.

**Retorno:** valor e booleano que indica existência.

**Exemplo isolado:**

```go
gitPath, exists := os.LookupEnv("PATH")
```

**Uso conceitual:** diagnosticar ambiente e entender como programas externos são localizados. Para verificar se o executável está disponível, estude também `exec.LookPath`.

## 6.2 `path/filepath` — caminhos do sistema de arquivos

Documentação: [pkg.go.dev/path/filepath](https://pkg.go.dev/path/filepath)

Este pacote trabalha com separadores e convenções de caminho do sistema operacional. Não confunda com `path`, que é voltado a caminhos com barra `/`, como URLs.

### `filepath.Abs`

**Assinatura:**

```go
func Abs(path string) (string, error)
```

**O que faz:** converte um caminho em uma representação absoluta.

**Entrada:** caminho relativo ou absoluto.

**Retorno:** caminho absoluto e erro.

**Exemplo isolado:**

```go
absolutePath, err := filepath.Abs("./meu-projeto")
```

### `filepath.Join`

**Assinatura:**

```go
func Join(elem ...string) string
```

**O que faz:** junta elementos de caminho usando as regras do sistema operacional e limpa o resultado.

**Entrada:** um ou mais elementos de caminho.

**Retorno:** caminho resultante.

**Exemplo isolado:**

```go
configPath := filepath.Join(homeDirectory, ".config", "gitdash")
```

### `filepath.Clean`

**Assinatura:**

```go
func Clean(path string) string
```

**O que faz:** simplifica lexicalmente um caminho, eliminando segmentos redundantes como `.` e resolvendo `..` quando possível.

**Entrada:** caminho.

**Retorno:** caminho normalizado lexicalmente.

**Importante:** `Clean` não verifica se o caminho existe e não resolve links simbólicos.

### `filepath.Rel`

**Assinatura:**

```go
func Rel(basePath, targPath string) (string, error)
```

**O que faz:** calcula um caminho relativo entre um caminho-base e um destino.

**Entrada:** caminho-base e caminho-alvo.

**Retorno:** caminho relativo e erro.

**Exemplo isolado:**

```go
relativePath, err := filepath.Rel(repositoryRoot, selectedFile)
```

**Uso conceitual:** exibir caminhos de arquivos de forma mais curta, relativa à raiz do repositório.

## 6.3 `os/exec` — execução de processos externos

Documentação: [pkg.go.dev/os/exec](https://pkg.go.dev/os/exec)

Este é um dos pacotes centrais do projeto.

### `exec.Command`

**Assinatura:**

```go
func Command(name string, arg ...string) *exec.Cmd
```

**O que faz:** prepara um processo para executar um programa com argumentos.

**Entrada:** nome/caminho do executável e argumentos separados.

**Retorno:** ponteiro para `exec.Cmd`. A chamada prepara o comando; a execução acontece ao chamar métodos como `Run`, `Output` ou `CombinedOutput`.

**Exemplo isolado:**

```go
cmd := exec.Command("git", "--version")
```

**Detalhe importante:** quando o nome não contém separador de diretório, Go procura o executável usando `PATH`. A função não inicia automaticamente um shell.

### `exec.CommandContext`

**Assinatura:**

```go
func CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd
```

**O que faz:** prepara um processo associado a um `context.Context`.

**Entrada:** contexto, executável e argumentos.

**Retorno:** `*exec.Cmd`.

**Uso conceitual:** permitir cancelar uma consulta Git quando ela deixa de ser necessária ou quando a aplicação está encerrando.

### `(*exec.Cmd).Output`

**Assinatura:**

```go
func (c *exec.Cmd) Output() ([]byte, error)
```

**O que faz:** executa o comando e retorna o `stdout`.

**Entrada:** nenhuma adicional; usa as configurações do `Cmd`.

**Retorno:** saída padrão em `[]byte` e erro. Se o processo terminar com status diferente de zero, normalmente o erro será um `*exec.ExitError`.

**Exemplo isolado:**

```go
output, err := cmd.Output()
```

**Ponto de estudo:** conversão de `[]byte` para `string`, tratamento de erro e acesso ao `stderr` armazenado em `ExitError` quando aplicável.

### `(*exec.Cmd).CombinedOutput`

**Assinatura:**

```go
func (c *exec.Cmd) CombinedOutput() ([]byte, error)
```

**O que faz:** executa o comando e combina `stdout` e `stderr` no mesmo retorno.

**Entrada:** nenhuma adicional.

**Retorno:** bytes combinados e erro.

**Quando estudar:** útil para diagnóstico de comandos em que a mensagem de erro do executável é importante. Não use uma saída combinada como se fosse um resultado limpo e estruturado sem considerar que mensagens de erro também podem estar presentes.

### `(*exec.Cmd).Run`

**Assinatura:**

```go
func (c *exec.Cmd) Run() error
```

**O que faz:** inicia o processo e aguarda sua conclusão.

**Entrada:** nenhuma adicional.

**Retorno:** erro ou `nil` quando a execução termina com sucesso.

**Quando estudar:** comandos nos quais a saída não é necessária, ou quando `Stdin`, `Stdout` e `Stderr` foram configurados separadamente.

### `exec.LookPath`

**Assinatura:**

```go
func LookPath(file string) (string, error)
```

**O que faz:** procura um executável pelo nome usando o `PATH` do ambiente.

**Entrada:** nome do executável.

**Retorno:** caminho encontrado e erro se não for localizado.

**Uso conceitual:** verificar a presença do Git antes de tentar consultar o repositório.

### `exec.ExitError`

**Tipo:**

```go
type ExitError struct { /* campos internos */ }
```

**O que representa:** processo que foi iniciado, mas terminou com status de saída diferente de zero.

**Ponto de estudo:** distinguir “Git não encontrado” de “Git encontrado, mas o comando falhou porque o diretório não é repositório”. São problemas diferentes e devem gerar mensagens diferentes.

### Regras para uso seguro

- Não reutilize a mesma instância de `Cmd` depois de chamar `Start`, `Run`, `Output` ou `CombinedOutput`.
- Não concatene argumentos em uma linha de shell.
- Defina diretório de execução explicitamente quando necessário, usando o campo `Cmd.Dir` ou o argumento `-C` do Git.
- Use contextos com prazo para operações que possam demorar.
- Capture e trate `stderr` quando for importante para diagnóstico.
- Não use `panic` para erros esperados de execução externa.

## 6.4 `context` — cancelamento e prazo

Documentação: [pkg.go.dev/context](https://pkg.go.dev/context)

### `context.WithCancel`

**Assinatura:**

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
```

**O que faz:** cria um contexto filho que pode ser cancelado explicitamente.

**Entrada:** contexto pai.

**Retorno:** contexto filho e função de cancelamento.

### `context.WithTimeout`

**Assinatura:**

```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
```

**O que faz:** cria um contexto que será cancelado quando o prazo expirar ou quando a função de cancelamento for chamada.

**Entrada:** contexto pai e duração máxima.

**Retorno:** contexto e função de cancelamento.

**Exemplo isolado:**

```go
ctx, cancel := context.WithTimeout(parent, 3*time.Second)
```

**Ponto de estudo:** a função `cancel` deve ser chamada para liberar recursos associados ao contexto. Leia também `Context.Done()`, `Context.Err()` e `context.Canceled` / `context.DeadlineExceeded`.

## 6.5 `os/signal` — sinais do processo

Documentação: [pkg.go.dev/os/signal](https://pkg.go.dev/os/signal)

### `signal.NotifyContext`

**Assinatura:**

```go
func NotifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc)
```

**O que faz:** cria um contexto cancelado quando um dos sinais indicados é recebido.

**Entrada:** contexto pai e sinais do sistema.

**Retorno:** contexto derivado e função para interromper a notificação.

**Uso conceitual:** compreender como o processo reage a interrupções do usuário e como o encerramento da aplicação se relaciona com o cancelamento de operações externas. O Bubble Tea já trata parte do ciclo de vida do terminal; não duplique mecanismos sem necessidade.

## 6.6 `strings` — tratamento de texto

Documentação: [pkg.go.dev/strings](https://pkg.go.dev/strings)

### `strings.TrimSpace`

**Assinatura:**

```go
func TrimSpace(s string) string
```

**O que faz:** remove espaços em branco do início e do final de uma string.

**Exemplo isolado:**

```go
branchName := strings.TrimSpace(rawOutput)
```

**Uso conceitual:** limpar newline final de saídas simples como o nome da branch.

### `strings.Split`

**Assinatura:**

```go
func Split(s, sep string) []string
```

**O que faz:** divide uma string em partes usando um separador literal.

**Retorno:** slice de strings.

**Atenção:** dividir saída Git por espaço não é uma estratégia segura para nomes de arquivos. `Split` é útil somente quando o formato realmente garante aquele delimitador.

### `strings.Cut`

**Assinatura:**

```go
func Cut(s, sep string) (before, after string, found bool)
```

**O que faz:** separa uma string na primeira ocorrência do separador.

**Retorno:** trecho anterior, trecho posterior e indicador de ocorrência.

**Uso conceitual:** interpretar campos simples com delimitador garantido. Não substitui um parser para formatos mais complexos.

## 6.7 `bufio` — leitura por linhas

Documentação: [pkg.go.dev/bufio](https://pkg.go.dev/bufio)

### `bufio.NewScanner`

**Assinatura:**

```go
func NewScanner(r io.Reader) *Scanner
```

**O que faz:** cria um scanner para percorrer uma fonte de dados por tokens; por padrão, o token é uma linha.

**Entrada:** qualquer `io.Reader`.

**Retorno:** `*bufio.Scanner`.

### `(*bufio.Scanner).Scan`

**Assinatura:**

```go
func (s *Scanner) Scan() bool
```

**O que faz:** avança para o próximo token.

**Retorno:** `true` quando um token foi obtido; `false` ao terminar ou ocorrer erro.

### `(*bufio.Scanner).Text`

**Assinatura:**

```go
func (s *Scanner) Text() string
```

**O que faz:** retorna o token atual como string.

### `(*bufio.Scanner).Err`

**Assinatura:**

```go
func (s *Scanner) Err() error
```

**O que faz:** retorna o erro ocorrido durante a leitura, se houver.

**Atenção:** o Scanner tem um limite padrão de tamanho de token. Para linhas maiores, estude `Scanner.Buffer` ou outras abstrações de leitura. Para parsing de nomes de arquivos delimitados por NUL, estude também `bytes` e leitura binária; não trate o formato como linhas comuns.

## 6.8 `errors` — inspeção de erros

Documentação: [pkg.go.dev/errors](https://pkg.go.dev/errors)

### `errors.Is`

**Assinatura:**

```go
func Is(err, target error) bool
```

**O que faz:** verifica se um erro corresponde a um erro-alvo, inclusive através de erros encapsulados.

### `errors.As`

**Assinatura:**

```go
func As(err error, target any) bool
```

**O que faz:** procura na cadeia de erros um valor que possa ser atribuído ao tipo apontado por `target`.

**Uso conceitual:** extrair informações de um `*exec.ExitError` sem depender de comparação textual da mensagem de erro.

## 6.9 `testing` — testes

Documentação: [pkg.go.dev/testing](https://pkg.go.dev/testing)

### `(*testing.T).TempDir`

**Assinatura:**

```go
func (t *T) TempDir() string
```

**O que faz:** cria um diretório temporário exclusivo para o teste. O diretório é removido ao final do teste e de seus subtestes.

**Uso conceitual:** criar repositórios Git temporários para testar diretórios válidos, inválidos e estados específicos.

### `(*testing.T).Run`

**Assinatura:**

```go
func (t *T) Run(name string, f func(t *T)) bool
```

**O que faz:** executa um subteste com nome próprio.

**Uso conceitual:** agrupar cenários como `repo limpo`, `repo com alterações`, `sem commit inicial` e `fora de repositório`.

### `(*testing.T).Helper`

**Assinatura:**

```go
func (t *T) Helper()
```

**O que faz:** marca uma função auxiliar como helper de teste para que informações de falha apontem para o chamador.

**Próximas leituras:** [Testes em Go](https://go.dev/doc/tutorial/add-a-test), [subtests e sub-benchmarks](https://go.dev/blog/subtests) e [race detector](https://go.dev/doc/articles/race_detector).

---

# 7. Ecossistema Charm — bibliotecas e quando considerar cada uma

O Charmbracelet mantém um conjunto de bibliotecas que se complementam. Nem todas precisam entrar no projeto. A regra é escolher uma dependência quando ela resolver uma necessidade real, não apenas porque existe.

## 7.1 Bubble Tea — núcleo da aplicação

**Import:** `charm.land/bubbletea/v2`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/bubbletea/v2) · [GitHub](https://github.com/charmbracelet/bubbletea)

É o framework responsável pelo ciclo de eventos e estado da TUI.

### APIs principais

#### `tea.NewProgram`

**Assinatura:**

```go
func NewProgram(model Model, opts ...ProgramOption) *Program
```

**O que faz:** cria um programa Bubble Tea a partir do model inicial e de opções.

**Entrada:** implementação de `tea.Model` e zero ou mais opções.

**Retorno:** `*tea.Program`.

#### `(*tea.Program).Run`

**Assinatura:**

```go
func (p *Program) Run() (returnModel Model, returnErr error)
```

**O que faz:** inicia o loop do programa e bloqueia até o encerramento.

**Retorno:** model final e erro de execução, quando houver.

#### `tea.Quit`

**Assinatura:**

```go
func Quit() Msg
```

**O que faz:** produz a mensagem especial que encerra o programa.

#### `tea.Batch`

**Assinatura:**

```go
func Batch(cmds ...Cmd) Cmd
```

**O que faz:** agenda vários comandos concorrentemente. Não garante a ordem de chegada dos resultados.

**Uso conceitual:** consultas independentes, como resumo do repositório e informações do último commit.

#### `tea.Sequence`

**Assinatura:**

```go
func Sequence(cmds ...Cmd) Cmd
```

**O que faz:** executa comandos em sequência, um após o outro.

**Diferença importante:** `Batch` é concorrente; `Sequence` respeita a ordem.

#### `tea.Tick`

**Assinatura:**

```go
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd
```

**O que faz:** agenda uma mensagem depois da duração indicada. O intervalo começa quando o comando é invocado.

**Uso conceitual:** temporizadores e atualização periódica, caso isso seja uma necessidade real. Não atualize o Git em alta frequência sem motivo.

#### `tea.WindowSizeMsg`

**Campos:** `Width int`, `Height int`.

**O que representa:** evento enviado inicialmente e quando o tamanho do terminal muda.

**Uso conceitual:** recalcular dimensões de painéis, tabelas e viewports.

#### `tea.KeyPressMsg`

**O que representa:** evento de tecla pressionada.

**Uso conceitual:** receber atalhos de navegação, atualização, abertura de detalhes e encerramento.

#### `tea.WithContext`

**Assinatura:**

```go
func WithContext(ctx context.Context) ProgramOption
```

**O que faz:** associa um contexto ao programa, permitindo cancelamento externo da execução.

### Leitura adicional

- [Tutorial oficial no repositório](https://github.com/charmbracelet/bubbletea/tree/main/tutorials)
- [Exemplos oficiais](https://github.com/charmbracelet/bubbletea/tree/main/examples)
- [API completa](https://pkg.go.dev/charm.land/bubbletea/v2)

Leia pelo menos um exemplo simples e um exemplo que execute I/O assíncrono. Observe a diferença entre a função que prepara um `Cmd` e a função que efetivamente executa a operação.

## 7.2 Bubbles — componentes prontos de TUI

**Import raiz:** `charm.land/bubbles/v2`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/bubbles/v2) · [GitHub](https://github.com/charmbracelet/bubbles)

Bubbles oferece componentes que implementam partes recorrentes de interfaces Bubble Tea.

### Componentes relevantes para o Git Dashboard

| Pacote | Finalidade | Possível utilização |
|---|---|---|
| `list` | Lista navegável com filtro, paginação e ajuda | Lista de arquivos ou commits |
| `table` | Tabela com linhas, colunas e seleção | Resumo tabular de arquivos |
| `viewport` | Área de conteúdo com rolagem | Diff ou detalhes longos |
| `help` | Ajuda de atalhos | Rodapé ou painel de ajuda |
| `key` | Definição de bindings | Centralizar atalhos |
| `spinner` | Indicador animado | Consulta Git em andamento |
| `textinput` | Campo de texto de uma linha | Busca ou escolha de caminho |
| `paginator` | Lógica e exibição de paginação | Histórico de commits |

Não é obrigatório usar todos. Uma lista e uma viewport podem ser suficientes para a primeira experiência visual.

### Pacote `table`

**Import:** `charm.land/bubbles/v2/table`  
**Documentação:** [pkg.go.dev/charm.land/bubbles/v2/table](https://pkg.go.dev/charm.land/bubbles/v2/table)

#### `table.New`

**Assinatura:**

```go
func New(opts ...Option) Model
```

Cria o model da tabela. As opções definem colunas, linhas, dimensões, foco e estilos.

#### `table.WithColumns`

**Assinatura:**

```go
func WithColumns(cols []Column) Option
```

Define as colunas e suas larguras/títulos.

#### `table.WithRows`

**Assinatura:**

```go
func WithRows(rows []Row) Option
```

Define as linhas. `Row` é um slice de strings.

#### `(*table.Model).SetRows`

**Assinatura:**

```go
func (m *Model) SetRows(r []Row)
```

Substitui o conjunto de linhas da tabela.

#### `(*table.Model).SetColumns`

**Assinatura:**

```go
func (m *Model) SetColumns(c []Column)
```

Substitui as colunas.

#### `table.Model.SelectedRow`

**Assinatura:**

```go
func (m Model) SelectedRow() Row
```

Retorna a linha selecionada.

#### `table.Model.Update` e `table.Model.View`

`Update` recebe uma mensagem Bubble Tea e retorna o model atualizado e um possível comando. `View` produz a representação textual do componente.

### Pacote `viewport`

**Import:** `charm.land/bubbles/v2/viewport`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/bubbles/v2/viewport)

Principais elementos:

- `viewport.New(opts ...Option) Model`: cria o model da viewport.
- `Model.SetContent(string)`: define o conteúdo que será exibido.
- `Model.SetWidth(int)` / `Model.SetHeight(int)`: ajustam dimensões.
- `Model.GotoTop()` / `Model.GotoBottom()`: movem a posição de rolagem.
- `Model.LineUp(int)` / `Model.LineDown(int)`: rolam por quantidade de linhas.
- `Model.View() string`: renderiza o conteúdo visível.
- `Model.Update(tea.Msg) (Model, tea.Cmd)`: processa eventos.

**Uso conceitual:** apresentar diffs longos sem tentar renderizar todo o conteúdo em uma área fixa.

### Pacote `textinput`

**Import:** `charm.land/bubbles/v2/textinput`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/bubbles/v2/textinput)

- `textinput.New() Model`: cria um campo de entrada com configurações padrão.
- `Model.SetValue(string)`: define o conteúdo.
- `Model.Value() string`: retorna o conteúdo digitado.
- `Model.Focus() tea.Cmd`: coloca o campo em foco e retorna um comando associado ao foco/cursor.
- `Model.Blur()`: remove o foco.
- `Model.Update(tea.Msg) (Model, tea.Cmd)`: processa a mensagem recebida.
- `Model.View() string`: renderiza o campo.

**Uso conceitual:** busca por nome de arquivo ou escolha de diretório. Não crie campo de entrada antes de existir um fluxo de busca ou seleção que o justifique.

### Pacote `spinner`

**Import:** `charm.land/bubbles/v2/spinner`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/bubbles/v2/spinner)

- `spinner.New(opts ...Option) Model`: cria um indicador.
- `Model.Tick() tea.Msg`: mensagem usada para avançar o frame.
- `Model.Update(tea.Msg) (Model, tea.Cmd)`: atualiza o indicador.
- `Model.View() string`: produz a representação atual.
- `spinner.Dot`, `spinner.Line`, entre outros: presets de animação.

**Uso conceitual:** deixar claro que uma consulta está ocorrendo. A animação precisa ser alimentada por mensagens/ticks, não por um loop bloqueante.

### Bubbles — exemplos

- [Repositório e catálogo de componentes](https://github.com/charmbracelet/bubbles)
- [Pacotes e documentação](https://pkg.go.dev/charm.land/bubbles/v2)
- [Exemplos oficiais de Bubble Tea](https://github.com/charmbracelet/bubbletea/tree/main/examples)

## 7.3 Lip Gloss — estilos e layout

**Import:** `charm.land/lipgloss/v2`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/lipgloss/v2) · [GitHub](https://github.com/charmbracelet/lipgloss)

Lip Gloss é responsável por estilo e composição visual. Ele não substitui o Bubble Tea.

### APIs principais

#### `lipgloss.NewStyle`

**Assinatura:**

```go
func NewStyle() Style
```

Cria um estilo vazio, que pode receber regras de cor, alinhamento, borda, margem, padding e dimensões.

#### `Style.Render`

**Assinatura:**

```go
func (s Style) Render(strs ...string) string
```

Aplica o estilo ao conteúdo e retorna uma string preparada para terminal.

#### `lipgloss.Width`

**Assinatura:**

```go
func Width(str string) int
```

Retorna a largura visual em células de terminal. Sequências ANSI são ignoradas e caracteres de largura dupla são considerados.

**Não substitua por `len(string)`** para calcular layout: bytes, runes e células de terminal são medidas diferentes.

#### `lipgloss.Height`

**Assinatura:**

```go
func Height(str string) int
```

Retorna a altura do texto em linhas.

#### `lipgloss.Size`

**Assinatura:**

```go
func Size(str string) (width, height int)
```

Retorna largura e altura em células/linhas.

#### `lipgloss.JoinHorizontal` e `lipgloss.JoinVertical`

Juntam blocos de texto potencialmente multilinha em uma composição horizontal ou vertical.

**Uso conceitual:** montar cabeçalho, painéis laterais, área de conteúdo e rodapé.

### Estudo recomendado

- Estude `Style`, `Color`, `Border`, `Padding`, `Margin`, `Width`, `Height`, `Align` e `Render`.
- Estude como a biblioteca mede texto com ANSI e Unicode.
- Não fixe dimensões sem levar em conta `tea.WindowSizeMsg`.

## 7.4 Huh — formulários e confirmações

**Import:** `charm.land/huh/v2`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/huh/v2) · [GitHub](https://github.com/charmbracelet/huh)

Huh é uma biblioteca de formulários e prompts interativos, integrável a Bubble Tea.

Possíveis utilizações futuras:

- Escolher um repositório em uma lista de caminhos recentes.
- Filtrar ou escolher uma branch.
- Solicitar texto para uma operação futura.
- Pedir confirmação explícita antes de uma ação destrutiva.

APIs relevantes:

- `huh.NewForm(groups ...*Group) *Form`: cria um formulário com grupos.
- `Form.GetString(key string) string`: obtém a resposta de um campo como string.
- `Form.GetInt(key string) int`: obtém resposta numérica.
- `Form.GetBool(key string) bool`: obtém resposta booleana.
- `Form.Errors() []error`: retorna erros atuais do formulário.

**Decisão de escopo:** Huh não é necessário para o MVP de leitura. É uma boa biblioteca para estudar quando existirem fluxos que realmente precisem de formulário ou confirmação.

## 7.5 Glamour — renderização de Markdown

**Import:** `charm.land/glamour/v2`  
**Documentação:** [pkg.go.dev](https://pkg.go.dev/charm.land/glamour/v2) · [GitHub](https://github.com/charmbracelet/glamour)

Glamour renderiza Markdown em terminais compatíveis com ANSI.

Possíveis extensões:

- Exibir corpos de mensagens de commit que contenham Markdown.
- Visualizar um README dentro do dashboard.
- Renderizar documentação ou descrições de releases.

APIs de entrada:

- `glamour.Render(input, style string) (string, error)`: renderiza uma string Markdown com um estilo.
- `glamour.NewTermRenderer(options ...TermRendererOption) (*TermRenderer, error)`: cria um renderer configurável.
- `(*TermRenderer).Render(input string) (string, error)`: renderiza usando o renderer criado.

Não é necessário para exibir diffs de texto puro.

## 7.6 Log — logging para diagnóstico

**Import:** `charm.land/log/v2`  
**Documentação:** [GitHub](https://github.com/charmbracelet/log)

Biblioteca de logging estruturado com níveis e saída legível. Pode ser útil para registrar falhas de execução Git e detalhes de diagnóstico.

Cuidados em TUI:

- Não escreva logs arbitrariamente no mesmo `stdout` controlado pela interface.
- Prefira um arquivo ou `stderr` conforme a configuração e o comportamento do Bubble Tea.
- Não exponha dados sensíveis do ambiente, tokens ou credenciais em logs.

É opcional; o pacote padrão `log` também pode resolver o começo do projeto.

## 7.7 Harmonica — animações com mola

**Repositório:** [github.com/charmbracelet/harmonica](https://github.com/charmbracelet/harmonica)

Biblioteca de animação baseada em física de molas. Pode ser interessante para animações de interface, transições e indicadores mais elaborados.

Não é necessária para o dashboard funcionar. Trate como estudo opcional depois de dominar mensagens, comandos, timers e atualização de componentes.

## 7.8 Gum, Fang e Wish — onde entram?

São projetos do ecossistema Charm, mas não são dependências necessárias para este dashboard:

- **[Gum](https://github.com/charmbracelet/gum):** conjunto de utilitários de shell para criar interações e scripts de terminal. É mais um produto/CLI para uso direto em shell do que um conjunto de componentes que o dashboard precisa incorporar.
- **[Fang](https://github.com/charmbracelet/fang):** starter kit para aplicações Cobra, com ajuda, erros e recursos de CLI. Pode ser útil se futuramente houver uma CLI com subcomandos e flags mais elaborados, mas não é requisito de uma TUI Bubble Tea simples.
- **[Wish](https://github.com/charmbracelet/wish):** toolkit para hospedar aplicações de terminal via SSH. Só faz sentido se o projeto evoluir para uma aplicação TUI remota.

A lista completa e o estado atual dos projetos podem ser explorados na organização [Charmbracelet no GitHub](https://github.com/charmbracelet).

---

# 8. Arquitetura lógica — responsabilidades esperadas

A organização final dos pacotes é uma decisão do desenvolvedor. Abaixo estão limites de responsabilidade, não uma árvore obrigatória de diretórios.

## 8.1 Entrada da aplicação

Responsabilidades:

- Iniciar o processo.
- Interpretar argumentos de execução, se houver.
- Determinar o diretório de trabalho inicial.
- Criar o estado inicial da aplicação.
- Criar e executar o programa Bubble Tea.
- Tratar erro de encerramento de maneira adequada.

## 8.2 Serviço de repositório Git

Responsabilidades:

- Executar o programa `git`.
- Definir o diretório correto do processo.
- Solicitar dados de repositório.
- Capturar stdout/stderr e exit code.
- Interpretar a saída.
- Converter dados em estruturas Go próprias do projeto.
- Retornar erros contextualizados.

Este serviço não deve importar Bubble Tea.

## 8.3 Domínio / dados

Conceitos de dados que podem existir:

### RepositoryInfo

Informações gerais: raiz do repositório, branch, hash atual, estado de detached HEAD.

### CommitInfo

Hash, subject, autor e data do commit.

### FileChange

Caminho, status no index, status no working tree, eventual caminho anterior em uma renomeação e categoria de alteração.

### RepositorySnapshot

Conjunto coerente das informações obtidas em uma atualização: metadados, último commit, status e lista de arquivos.

### GitError

Erro contextualizado com operação, diretório e causa original. Evite reduzir todo erro a uma string, pois a interface pode precisar diferenciar suas categorias.

Esses nomes são sugestões de vocabulário. O modelo final deve refletir o entendimento que você construir sobre o domínio.

## 8.4 Model da TUI

Responsabilidades:

- Guardar o snapshot atual.
- Guardar seleção e painel ativo.
- Guardar dimensões do terminal.
- Guardar loading/erro.
- Interpretar eventos e mensagens.
- Solicitar comandos assíncronos.
- Renderizar os componentes.

Não deve executar parsing complexo de Git dentro do método `View`.

## 8.5 Componentes visuais

Cada componente visual deve ter uma função clara. Não transforme cada label em um componente independente sem necessidade, mas evite concentrar tabela, diff, ajuda, navegação e lógica de domínio em um único bloco gigantesco.

---

# 9. Concorrência e I/O assíncrono

## 9.1 Por que isso importa

Executar um processo externo pode demorar. Um repositório grande, filesystem lento, hooks configurados ou um processo Git preso podem atrasar a resposta.

A TUI precisa continuar recebendo eventos e redesenhando a tela durante consultas demoradas.

## 9.2 Relação entre Go e Bubble Tea

Go oferece goroutines e channels como primitivas de concorrência. Bubble Tea oferece `tea.Cmd` e `tea.Msg` como convenção de I/O e retorno ao loop de interface.

Não é necessário criar uma goroutine manual para cada tecla. Para operações que se encaixam no ciclo de Bubble Tea, use o modelo de comandos e mensagens.

## 9.3 Cuidados necessários

- Não bloquear `Update` com operações demoradas.
- Não executar processos Git dentro de `View`.
- Definir política para duas atualizações solicitadas rapidamente.
- Evitar que uma resposta antiga sobrescreva uma atualização mais recente.
- Considerar cancelamento de uma consulta em andamento.
- Não compartilhar mapas, slices ou structs mutáveis entre goroutines sem sincronização ou transferência clara de ownership.
- Usar `tea.Batch` somente quando as operações forem independentes.
- Não usar `tea.Sequence` quando a ordem não tiver significado — e não usar `Batch` quando a segunda operação depender do resultado da primeira.

## 9.4 Tópicos Go para estudar

- [A Tour of Go — Concurrency](https://go.dev/tour/concurrency/1)
- [Effective Go — Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Go Blog — Share Memory By Communicating](https://go.dev/doc/effective_go#sharing)
- [Race Detector](https://go.dev/doc/articles/race_detector)
- [Package context](https://pkg.go.dev/context)

---

# 10. Tratamento de erros e observabilidade

## 10.1 Categorias de erro

A aplicação deve distinguir, pelo menos:

| Categoria | Exemplo | Tratamento esperado |
|---|---|---|
| Caminho inválido | Diretório inexistente | Mensagem clara e possibilidade de tentar outro caminho |
| Permissão | Sem acesso ao diretório | Exibir erro contextualizado |
| Git ausente | Executável não localizado | Explicar que Git precisa estar instalado/disponível no PATH |
| Não é repositório | `rev-parse` falha no diretório | Informar que o diretório não é uma working tree Git |
| Sem commit inicial | `HEAD` ainda não aponta para commit | Mostrar estado especial, sem tratar como corrupção |
| Comando Git falhou | Exit code diferente de zero | Preservar stderr e contexto da operação |
| Cancelamento | Contexto cancelado | Não exibir como falha inesperada |
| Parsing inválido | Saída não corresponde ao formato esperado | Informar erro de leitura e preservar contexto para diagnóstico |

## 10.2 Princípios

- Erro não é a mesma coisa que estado vazio.
- Não descarte a causa original do erro.
- Não dependa do texto exato da mensagem de erro do Git para identificar todos os cenários.
- Use tipos e categorias quando a aplicação precisar tomar decisões diferentes.
- Mensagens para usuário devem ser simples; detalhes técnicos podem ir para logs.

---

# 11. Testes e critérios de qualidade

## 11.1 Estratégia de testes

### Testes unitários

Testar parsing e transformação de saída sem iniciar Bubble Tea e, sempre que possível, sem depender de um repositório real.

Entradas importantes:

- Status limpo.
- Arquivo modificado staged.
- Arquivo modificado unstaged.
- Arquivo untracked.
- Arquivo removido.
- Conflito.
- Renomeação.
- Caminho com espaços e Unicode.
- Saída vazia.
- Saída malformada.

### Testes de integração

Usar diretórios temporários para criar repositórios de teste com o Git instalado.

Cenários:

- Repositório recém-inicializado.
- Repositório com commit inicial.
- Repositório limpo.
- Alteração unstaged.
- Alteração staged.
- Arquivo untracked.
- Arquivo removido.
- Repositório com mais de um commit.
- Execução a partir de subdiretório.
- Detached HEAD.

### Testes de interface

Verificar principalmente:

- Atualização do estado ao receber mensagens.
- Navegação da lista.
- Seleção e troca do painel.
- Estado de carregamento.
- Erros e estados especiais.
- Redimensionamento do terminal.
- Encerramento correto.

## 11.2 Critérios de aceite do MVP

O MVP pode ser considerado funcional quando:

- [ ] Identifica corretamente a raiz de um repositório a partir do diretório atual ou de um caminho informado.
- [ ] Diferencia diretório inválido, diretório fora de repositório e Git ausente.
- [ ] Exibe branch ou estado `detached HEAD`.
- [ ] Exibe hash e subject do commit atual quando disponíveis.
- [ ] Exibe estado limpo sem apresentar erro.
- [ ] Diferencia alterações staged e unstaged.
- [ ] Identifica arquivos untracked.
- [ ] Permite navegar pela lista de arquivos.
- [ ] Exibe o diff de um arquivo compatível.
- [ ] Atualiza dados sem congelar a interface.
- [ ] Reage ao redimensionamento do terminal.
- [ ] Trata nomes com espaços e Unicode.
- [ ] Não executa operações que alterem o repositório.
- [ ] Possui testes de parsing e testes de integração dos cenários essenciais.

## 11.3 Ferramentas Go

Comandos úteis para conhecer e usar ao longo do projeto:

| Comando | Finalidade |
|---|---|
| `go mod init <module>` | Inicializar o módulo |
| `go get <module>` | Adicionar/atualizar dependência |
| `go mod tidy` | Sincronizar dependências com os imports |
| `go run .` | Executar o programa do módulo/pacote atual |
| `go test ./...` | Executar testes de todos os pacotes |
| `go test -race ./...` | Executar testes com detector de race |
| `go vet ./...` | Executar análises estáticas incluídas no vet |
| `go doc <package>` | Consultar documentação de pacote no terminal |
| `go list -m all` | Listar módulos envolvidos |

Não é necessário memorizar todos. Consulte `go help` e a documentação oficial quando precisar.

---

# 12. Roadmap sugerido por entregas

A ordem abaixo organiza o risco técnico. Não é um roteiro de código linha por linha.

## Entrega A — Exploração do ambiente

Resultado esperado: entender como Go representa caminhos, consulta o diretório atual, verifica diretórios e executa processos externos.

Evidências de domínio:

- Saber explicar a diferença entre `os.Getwd()` e executar o comando `pwd`.
- Saber explicar a diferença entre preparar um `exec.Cmd` e executá-lo.
- Saber dizer onde ficam `stdout`, `stderr` e exit code.
- Saber lidar com caminho com espaços sem recorrer a shell.

## Entrega B — Inspeção Git sem interface

Resultado esperado: conseguir obter as informações do repositório e entender a saída de cada comando.

Evidências de domínio:

- Reconhecer a raiz do repositório.
- Tratar branch vazia em `detached HEAD`.
- Diferenciar estado staged e unstaged.
- Entender porcelain e a dificuldade de parsing de caminhos.
- Entender por que um repositório sem commit inicial é um caso especial.

## Entrega C — Modelo de domínio e testes

Resultado esperado: representar as informações Git em tipos Go próprios e validar parsing/erros.

Evidências de domínio:

- Separar saída bruta e dados interpretados.
- Explicar o significado de cada campo do status.
- Testar entradas válidas, inválidas e casos extremos.
- Diferenciar erro operacional de ausência de dados.

## Entrega D — Primeira TUI

Resultado esperado: uma TUI mínima com estado, teclado, renderização e encerramento corretos.

Evidências de domínio:

- Explicar Model, Msg e Cmd.
- Explicar o papel de `Init`, `Update` e `View`.
- Explicar por que uma operação Git não deve bloquear a renderização.
- Entender o contrato de `tea.View` no Bubble Tea v2.

## Entrega E — Dashboard integrado

Resultado esperado: ligar o serviço Git ao Bubble Tea e exibir as informações essenciais.

Evidências de domínio:

- Resultados assíncronos chegam como mensagens.
- Loading, sucesso e erro têm estados próprios.
- A seleção da lista determina o conteúdo do painel de detalhes.
- Atualização não deixa respostas antigas sobrescreverem estado novo.

## Entrega F — Refinamento visual

Resultado esperado: interface organizada e utilizável em tamanhos diferentes de terminal.

Evidências de domínio:

- Usar Bubbles sem deixar de entender o model do componente.
- Usar Lip Gloss para estilo e composição.
- Medir células com utilitários apropriados, sem confiar em `len` para largura visual.
- Tratar diffs longos com viewport.

## Entrega G — Robustez

Resultado esperado: testes de integração, mensagens de erro compreensíveis e comportamento estável em situações incomuns.

Evidências de domínio:

- Testar repositórios temporários.
- Testar nomes de arquivos incomuns.
- Testar diretórios inválidos e Git ausente.
- Testar terminal pequeno e redimensionamento.
- Executar `go test -race ./...`.

---

# 13. Referências de Go — trilha oficial

A sequência abaixo prioriza documentação oficial e material de referência durável.

## Começo e linguagem

1. [Go Documentation](https://go.dev/doc/) — portal central da documentação.
2. [Get started with Go](https://go.dev/doc/tutorial/getting-started) — ferramentas, pacotes e primeiro programa.
3. [A Tour of Go](https://go.dev/tour/) — sintaxe, tipos, métodos, interfaces e concorrência.
4. [Create a Go module](https://go.dev/doc/tutorial/create-module) — módulos, erros, slices, maps, testes e build.
5. [Effective Go](https://go.dev/doc/effective_go) — estilo idiomático e organização de código.
6. [Go Language Specification](https://go.dev/ref/spec) — referência formal da linguagem.

## Módulos e dependências

- [Go Modules Reference](https://go.dev/ref/mod)
- [Managing dependencies](https://go.dev/doc/modules/managing-dependencies)
- [How to write Go code](https://go.dev/doc/code)

## Biblioteca padrão

- [`os`](https://pkg.go.dev/os)
- [`os/exec`](https://pkg.go.dev/os/exec)
- [`path/filepath`](https://pkg.go.dev/path/filepath)
- [`context`](https://pkg.go.dev/context)
- [`os/signal`](https://pkg.go.dev/os/signal)
- [`strings`](https://pkg.go.dev/strings)
- [`bytes`](https://pkg.go.dev/bytes)
- [`bufio`](https://pkg.go.dev/bufio)
- [`errors`](https://pkg.go.dev/errors)
- [`testing`](https://pkg.go.dev/testing)
- [`time`](https://pkg.go.dev/time)

## Qualidade e concorrência

- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
- [Data Race Detector](https://go.dev/doc/articles/race_detector)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go Blog](https://go.dev/blog/)

---

# 14. Referências Charm — trilha oficial

| Projeto | Import path atual documentado | Papel no dashboard | Prioridade |
|---|---|---|---|
| Bubble Tea | `charm.land/bubbletea/v2` | Loop de estado/eventos | Essencial |
| Bubbles | `charm.land/bubbles/v2/...` | Componentes TUI | Alta |
| Lip Gloss | `charm.land/lipgloss/v2` | Estilos e layout | Alta |
| Huh | `charm.land/huh/v2` | Formulários/confirmações futuras | Opcional |
| Glamour | `charm.land/glamour/v2` | Markdown no terminal | Opcional |
| Log | `charm.land/log/v2` | Logging | Opcional |
| Harmonica | consultar o repositório | Animações | Opcional |
| Gum | CLI independente | Scripts e UX de shell | Apenas referência |
| Fang | CLI/Cobra | Estrutura de CLI tradicional | Apenas referência |
| Wish | servidor SSH para TUIs | Execução remota | Apenas referência |

### Links gerais

- [Organização Charmbracelet](https://github.com/charmbracelet)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Bubbles](https://github.com/charmbracelet/bubbles)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Huh](https://github.com/charmbracelet/huh)
- [Glamour](https://github.com/charmbracelet/glamour)
- [Log](https://github.com/charmbracelet/log)
- [Harmonica](https://github.com/charmbracelet/harmonica)
- [Gum](https://github.com/charmbracelet/gum)
- [Fang](https://github.com/charmbracelet/fang)
- [Wish](https://github.com/charmbracelet/wish)

**Nota de versão:** as versões e import paths dos projetos Charm podem mudar. Antes de adicionar uma dependência, confira `go get` e a página atual do módulo no `pkg.go.dev`. Não copie import paths de tutoriais antigos sem verificar se são v1 ou v2.

---

# 15. Glossário rápido

| Termo | Significado neste projeto |
|---|---|
| CLI | Aplicação operada por linha de comando |
| TUI | Interface interativa de terminal |
| Working tree | Arquivos e diretórios visíveis no checkout atual |
| Index / staging area | Área que guarda o conteúdo preparado para o próximo commit |
| `HEAD` | Referência que identifica o commit/posição atual do repositório |
| Branch | Referência móvel para uma linha de desenvolvimento |
| Detached HEAD | Estado em que `HEAD` aponta diretamente para um commit, sem branch atual selecionada |
| Untracked | Arquivo ainda não rastreado pelo Git |
| Staged | Alteração incluída no index para o próximo commit |
| Unstaged | Alteração no working tree que ainda não foi incluída no index |
| Diff | Representação das diferenças entre duas versões/estados |
| Model | Estado da aplicação Bubble Tea |
| Msg | Evento/mensagem recebido pelo Bubble Tea |
| Cmd | Operação de I/O que retorna uma mensagem |
| Component | Parte reutilizável da interface, como table ou viewport |
| Exit code | Código numérico retornado por um processo ao terminar |
| `stdout` | Saída padrão de um processo |
| `stderr` | Canal padrão de mensagens de erro/diagnóstico |
| Context | Mecanismo Go para cancelamento, prazo e propagação de escopo de operação |

---

# 16. Perguntas que o desenvolvedor deve conseguir responder

Estas perguntas servem como autoavaliação. Não precisam ser respondidas dentro do código nem enviadas a terceiros.

## Go e sistema operacional

1. Qual é a diferença entre `os.Getwd()` e executar `pwd` com `os/exec`?
2. Qual é a diferença entre criar um `exec.Cmd` e executar `Output()`?
3. Por que os argumentos de `exec.Command` devem ser passados separadamente?
4. Qual é a diferença entre `Output()` e `CombinedOutput()`?
5. Como diferenciar um erro ao localizar o programa de um exit code não zero do programa?
6. Por que um contexto com timeout é relevante para subprocessos?
7. Qual é a diferença entre bytes, runes e largura visual em células de terminal?

## Git

1. Por que consultar `git rev-parse --show-toplevel` é mais confiável do que verificar apenas se existe `.git`?
2. O que significa cada uma das duas posições do status `XY`?
3. Por que `git status` em formato humano não é ideal para parsing?
4. Por que a opção `-z` ajuda a tratar nomes de arquivo incomuns?
5. O que acontece com `git branch --show-current` em `detached HEAD`?
6. Por que `git rev-parse --short HEAD` pode falhar em um repositório recém-inicializado?
7. Qual é a diferença entre `git diff` e `git diff --cached`?

## Bubble Tea e Charm

1. Qual é a responsabilidade de `Init`, `Update` e `View`?
2. O que é um `tea.Cmd` e como seu resultado volta ao Model?
3. Qual é a diferença entre `tea.Batch` e `tea.Sequence`?
4. Por que operações de I/O não devem acontecer em `View`?
5. O que mudou no contrato de `View` na versão 2?
6. Qual componente Bubbles atende melhor uma lista de arquivos e qual atende melhor um diff extenso?
7. Por que `lipgloss.Width` é mais apropriado do que `len` para dimensionar texto de terminal?

---

## Definição de pronto do documento

O projeto não está pronto por simplesmente compilar ou mostrar uma tela colorida. O objetivo de aprendizado é conseguir explicar as decisões, justificar as bibliotecas escolhidas e entender o caminho completo dos dados:

**filesystem / processo Git → saída bruta → parsing → estruturas Go → mensagem Bubble Tea → estado do Model → View → terminal.**

A implementação é sua. Este documento é o contrato inicial do produto e o mapa de documentação para construí-lo.
