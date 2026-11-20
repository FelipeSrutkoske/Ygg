# go-vit

**go-vit** é um gerenciador visual de Git para o terminal (TUI), feito em Go, pensado para ser limpo, intuitivo e amigável tanto para iniciantes quanto para o uso diário.

## Instalação

Certifique-se de ter o **Go** e o **Git** instalados no sistema e execute:

```bash
go install github.com/FelipeSrutkoske/go-vit@latest
```

## Uso

Execute no diretório do repositório desejado ou informe o caminho como argumento:

```bash
# Abre o repositório no diretório atual
go-vit

# Abre um repositório em outro caminho
go-vit /caminho/do/repositorio
```

Opções e comandos disponíveis:

- `update`, `upgrade`: Atualiza o `go-vit` para a versão mais recente via `go install`
- `-h`, `--help`: Exibe a mensagem de ajuda
- `-v`, `--version`: Exibe a versão instalada

## Principais Funcionalidades

- **Branches**: Liste branches locais e remotas, veja indicadores de *upstream* e *ahead/behind*, crie novas branches, troque de branch (com opção automática de *Stash + Trocar* quando houver alterações pendentes), faça merge e delete branches com segurança.
- **Alterações**: Visualize arquivos modificados, adicionados, removidos e não rastreados; faça *stage* e *unstage*, crie commits, use *stash*/*pop* e descarte alterações com confirmação.
- **Histórico de Commits**: Navegue pelos commits da branch atual (ou da branch selecionada), inspecione o diff completo de cada commit e execute *revert* ou *cherry-pick* com confirmação.
- **Visualizador de Diff estilo VS Code**: Inspecione alterações em verde e vermelho com numeração de linhas, contagem `+X / -Y` e tela cheia alternando entre modo **Lado a Lado** e **Unificado** (`l`).
- **Filtro rápido (`/`)**: Filtre branches, arquivos alterados ou commits em tempo real em qualquer aba.
- **Preview de comandos Git**: Veja no rodapé o comando `git` exato executado por cada ação na interface.
- **Sincronização remota (`fetch`, `pull`, `push`)**: Envie commits com `P` (configurando automaticamente `git push -u origin <branch>` quando não houver *upstream*).
- **Conflitos**: Identifique blocos de conflito (`<<<<<<<` / `=======` / `>>>>>>>`) destacados visualmente e resolva escolhendo suas alterações (*Ours*), as alterações recebidas (*Theirs*) ou abrindo o arquivo diretamente no `$EDITOR`.

## Atalhos de Teclado

### Globais

| Atalho | Ação |
| --- | --- |
| `Tab` / `1` / `2` / `3` | Alternar entre **Branches**, **Alterações** e **Histórico** |
| `↑` / `k` e `↓` / `j` | Navegar pelos itens da lista |
| `g` / `Home` e `G` / `End` | Ir para o início ou fim da lista |
| `K` / `PgUp` / `Ctrl+U` e `J` / `PgDn` / `Ctrl+D` | Rolar o painel direito (commits ou diff) |
| `/` | Filtrar itens da aba atual (`Enter` fixa o filtro, `Esc` limpa) |
| `f` | Executar `git fetch --prune` |
| `p` | Executar `git pull` |
| `P` | Executar `git push` (configura `-u origin <branch>` se necessário) |
| `R` | Atualizar informações do repositório |
| `?` | Exibir / ocultar tela de ajuda |
| `q` / `Ctrl+C` | Sair do go-vit |

### Aba Branches (`1`)

| Atalho | Ação |
| --- | --- |
| `Enter` | Trocar para a branch selecionada |
| `r` | Mostrar / ocultar branches remotas |
| `n` | Criar nova branch |
| `d` | Deletar branch selecionada |
| `m` | Fazer merge da branch selecionada na branch atual |

### Aba Alterações & Conflitos (`2`)

| Atalho | Ação |
| --- | --- |
| `Espaço` / `s` | Adicionar ou remover arquivo do *stage* (*Stage / Unstage*) |
| `Shift+Espaço` / `S` | Alternar *stage*/*unstage* de todas as alterações (`git add -A` / `git reset HEAD`) |
| `Enter` / `d` | Abrir visualizador de diff em tela cheia (estilo VS Code) |
| `c` | Abrir modal de commit |
| `z` | Guardar (*Stash*) ou restaurar (*Pop*) alterações |
| `x` | Descartar alterações do arquivo (com confirmação) |

### Aba Histórico (`3`)

| Atalho | Ação |
| --- | --- |
| `Enter` / `d` | Abrir diff completo do commit em tela cheia (estilo VS Code) |
| `v` | Reverter commit selecionado (`git revert --no-edit`, com confirmação) |
| `y` | Aplicar commit na branch atual (`git cherry-pick`, com confirmação) |
| `b` | Alternar histórico entre `HEAD` atual e a branch selecionada na aba `1` |

### Visualizador de Diff em Tela Cheia (`Enter` / `d`)

| Atalho | Ação |
| --- | --- |
| `l` | Alternar entre modo **Lado a Lado** e **Unificado** |
| `j` / `k` e `J` / `K` | Rolar linha a linha ou meia página |
| `[` / `]` | Ir para o arquivo ou commit anterior / próximo |
| `Espaço` / `s` | Fazer *Stage / Unstage* do arquivo atual (na aba Alterações) |
| `e` | Abrir arquivo atual no `$EDITOR` |
| `Esc` / `q` / `Enter` | Fechar visualizador |

### Resolução de Conflitos

| Atalho | Ação |
| --- | --- |
| `1` | Manter nossas alterações (*Ours*) |
| `2` | Aceitar alterações recebidas (*Theirs*) |
| `e` | Abrir arquivo conflitado no `$EDITOR` |

## Estrutura do Projeto

```text
.
├── main.go              # Ponto de entrada e parsing de flags/argumentos
├── internal/
│   ├── git/             # Cliente Git via os/exec e parsers de status/diff
│   └── tui/             # Interface terminal em Bubble Tea v2 e Lip Gloss v2
├── go.mod
└── README.md
```

## Contribuição

Contribuições são bem-vindas. Para colaborar:

1. Faça um fork do repositório.
2. Crie uma branch para sua alteração (`git checkout -b minha-alteracao`).
3. Execute os testes antes de abrir o Pull Request:

```bash
go test -race ./...
go vet ./...
```

4. Abra um Pull Request descrevendo o problema resolvido ou a melhoria proposta.

## Privacidade e Uso de IA

Ao usar a geração de mensagem de commit com IA (`Ctrl+G`), o `git diff --cached` das alterações preparadas é enviado para os servidores da NVIDIA (`integrate.api.nvidia.com`). **Não utilize esta funcionalidade em repositórios com segredos expostos no diff ou sob NDA estrito.**

