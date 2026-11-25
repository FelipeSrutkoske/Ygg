package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/Ygg/internal/config"
	"github.com/FelipeSrutkoske/Ygg/internal/git"
	"github.com/FelipeSrutkoske/Ygg/internal/tui"
)

var version = "0.1.0"

const moduleInstallTarget = "github.com/FelipeSrutkoske/Ygg@latest"

func printHelp() {
	fmt.Print(`Ygg (Yggdrasil) — Gerenciador visual de Git no terminal

Uso:
  ygg [caminho]
  ygg [comando]
  ygg [opções]

Comandos:
  config           Verifica ou configura a chave de API de IA
  update, upgrade  Atualiza o ygg para a versão mais recente

Argumentos:
  caminho          Diretório do repositório Git (padrão: ".")

Opções:
  -h, --help       Exibe esta ajuda
  -v, --version    Exibe a versão do ygg
`)
}

func runUpdate() error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("o Go não foi encontrado no PATH: %w\nInstale o Go (https://go.dev/dl/) para atualizar via 'go install'", err)
	}

	fmt.Printf("Atualizando ygg (%s)...\n", moduleInstallTarget)
	cmd := exec.Command("go", "install", moduleInstallTarget)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao atualizar o ygg: %w", err)
	}

	fmt.Println("ygg atualizado com sucesso!")
	return nil
}

func runConfig(in io.Reader, out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}
	path, _ := config.ConfigPath()
	reader := bufio.NewReader(in)

	if strings.TrimSpace(cfg.AIAPIKey) != "" {
		fmt.Fprintf(out, "Chave de API atual: %s\n", config.MaskKey(cfg.AIAPIKey))
		if path != "" {
			fmt.Fprintf(out, "Arquivo de configuração: %s\n", path)
		}
		fmt.Fprint(out, "Deseja substituir a chave de API? (s/N): ")
		ans, _ := reader.ReadString('\n')
		ans = strings.ToLower(strings.TrimSpace(ans))
		if ans != "s" && ans != "sim" && ans != "y" && ans != "yes" {
			fmt.Fprintln(out, "Nenhuma alteração realizada.")
			return nil
		}
	} else {
		fmt.Fprintln(out, "Nenhuma chave de API de IA configurada.")
	}

	fmt.Fprint(out, "Cole sua chave de API: ")
	keyLine, err := reader.ReadString('\n')
	if err != nil && len(strings.TrimSpace(keyLine)) == 0 {
		return fmt.Errorf("leitura cancelada")
	}
	apiKey := strings.TrimSpace(keyLine)
	if apiKey == "" {
		return fmt.Errorf("a chave de API não pode estar vazia")
	}

	cfg.AIAPIKey = apiKey
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("erro ao salvar configuração: %w", err)
	}

	fmt.Fprintf(out, "Chave de API salva (%s)", config.MaskKey(apiKey))
	if path != "" {
		fmt.Fprintf(out, " em %s", path)
	}
	fmt.Fprintln(out, ".")
	return nil
}

func main() {
	targetDir := "."
	var hasDir bool

	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			printHelp()
			return
		case "-v", "--version":
			fmt.Printf("ygg v%s\n", version)
			return
		case "config":
			if len(os.Args) > 2 {
				fmt.Fprintf(os.Stderr, "Uso inválido: '%s' não aceita argumentos extras.\n", arg)
				os.Exit(1)
			}
			if err := runConfig(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
				os.Exit(1)
			}
			return
		case "update", "upgrade":
			if len(os.Args) > 2 {
				fmt.Fprintf(os.Stderr, "Uso inválido: '%s' não aceita argumentos extras.\n", arg)
				os.Exit(1)
			}
			if err := runUpdate(); err != nil {
				fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
				os.Exit(1)
			}
			return
		default:
			if len(arg) > 0 && arg[0] == '-' {
				fmt.Fprintf(os.Stderr, "Opção desconhecida: %s\nUse 'ygg --help' para ver as opções disponíveis.\n", arg)
				os.Exit(1)
			}
			if hasDir {
				fmt.Fprintln(os.Stderr, "Uso inválido: informe apenas um caminho de diretório.\nUse 'ygg --help' para mais informações.")
				os.Exit(1)
			}
			targetDir = arg
			hasDir = true
		}
	}

	info, err := os.Stat(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao acessar o diretório %q: %v\n", targetDir, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "O caminho %q não é um diretório válido.\n", targetDir)
		os.Exit(1)
	}

	if err := git.CheckGitInstalled(); err != nil {
		fmt.Fprintln(os.Stderr, "Erro: o Git não foi encontrado no sistema.")
		fmt.Fprintln(os.Stderr, "Instale o Git (https://git-scm.com/downloads) e tente novamente.")
		os.Exit(1)
	}

	client := git.NewClient(targetDir)
	m := tui.NewModel(client)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar o ygg: %v\n", err)
		os.Exit(1)
	}
}
