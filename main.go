package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/FelipeSrutkoske/go-vit/internal/git"
	"github.com/FelipeSrutkoske/go-vit/internal/tui"
)

var version = "0.1.0"

const moduleInstallTarget = "github.com/FelipeSrutkoske/go-vit@latest"

func printHelp() {
	fmt.Print(`go-vit — Gerenciador visual de Git no terminal

Uso:
  go-vit [caminho]
  go-vit [comando]
  go-vit [opções]

Comandos:
  update, upgrade  Atualiza o go-vit para a versão mais recente

Argumentos:
  caminho          Diretório do repositório Git (padrão: ".")

Opções:
  -h, --help       Exibe esta ajuda
  -v, --version    Exibe a versão do go-vit
`)
}

func runUpdate() error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("o Go não foi encontrado no PATH: %w\nInstale o Go (https://go.dev/dl/) para atualizar via 'go install'", err)
	}

	fmt.Printf("Atualizando go-vit (%s)...\n", moduleInstallTarget)
	cmd := exec.Command("go", "install", moduleInstallTarget)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao atualizar o go-vit: %w", err)
	}

	fmt.Println("go-vit atualizado com sucesso!")
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
			fmt.Printf("go-vit v%s\n", version)
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
				fmt.Fprintf(os.Stderr, "Opção desconhecida: %s\nUse 'go-vit --help' para ver as opções disponíveis.\n", arg)
				os.Exit(1)
			}
			if hasDir {
				fmt.Fprintln(os.Stderr, "Uso inválido: informe apenas um caminho de diretório.\nUse 'go-vit --help' para mais informações.")
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
		fmt.Fprintf(os.Stderr, "Erro ao executar o go-vit: %v\n", err)
		os.Exit(1)
	}
}
