package bubblecode

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
)

var rootCmd = &cobra.Command{
	Use:   "bubblecode",
	Short: "TUI for AI agents via the ACP protocol",
	Long: `A terminal user interface for interacting with AI agents via the ACP protocol.
Built with Bubble Tea v2 and Lip Gloss v2.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChat(cmd)
	},
}

func chatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Start the chat TUI (default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat(cmd)
		},
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(chatCmd())
	rootCmd.AddCommand(acpCmd())
	rootCmd.AddCommand(modelsCmd())
	rootCmd.AddCommand(providersCmd())
}

// ─── Models ────────────────────────────────────────────────────────────────

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Show or switch configured models",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := agent.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := agent.LoadConfig(path)
			if err != nil {
				fmt.Println("No config found. Use 'bubblecode providers config' to set up.")
				return nil
			}
			var n int
			for _, p := range cfg.Providers {
				for _, m := range p.Models {
					n++
					mm := m
					if p.Name == cfg.ActiveProvider && m == cfg.ActiveModel {
						mm += " *"
					}
					fmt.Printf("%2d. %s/%s\n", n, p.Name, mm)
				}
			}
			return nil
		},
	}
	cmd.AddCommand(modelsSwitchCmd())
	return cmd
}

func modelsSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <model>",
		Short: "Switch active model for current provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadProviders()
			if err != nil {
				return err
			}
			model := args[0]
			p := cfg.GetActiveProvider()
			if p == nil {
				return fmt.Errorf("no active provider")
			}
			cfg.ActiveModel = model
			if err := agent.SaveConfig(path, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Model switched to: %s\n", model)
			return nil
		},
	}
}

// ─── Providers ─────────────────────────────────────────────────────────────

func providersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage provider configurations",
	}
	cmd.AddCommand(providersListCmd())
	cmd.AddCommand(providersAddCmd())
	cmd.AddCommand(providersDeleteCmd())
	cmd.AddCommand(providersConfigCmd())
	return cmd
}

func loadProviders() (*agent.Config, string, error) {
	path, err := agent.ConfigPath()
	if err != nil {
		return nil, "", err
	}
	cfg, err := agent.LoadConfig(path)
	if err != nil {
		return nil, "", fmt.Errorf("config not found, run 'bubblecode providers config' first")
	}
	return cfg, path, nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func resolveProvider(cfg *agent.Config, target string) *agent.Provider {
	for i := range cfg.Providers {
		if cfg.Providers[i].Name == target {
			return &cfg.Providers[i]
		}
	}
	if idx, err := strconv.Atoi(target); err == nil && idx >= 1 && idx <= len(cfg.Providers) {
		return &cfg.Providers[idx-1]
	}
	return nil
}

func providersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := agent.ConfigPath()
			cfg, err := agent.LoadConfig(path)
			if err != nil || len(cfg.Providers) == 0 {
				fmt.Println("No providers configured. Use 'bubblecode providers add' or 'bubblecode providers config'.")
				return nil
			}
			fmt.Printf("Active: %s / %s\n\n", cfg.ActiveProvider, cfg.ActiveModel)
			for i, p := range cfg.Providers {
				mark := " "
				if p.Name == cfg.ActiveProvider {
					mark = ">"
				}
				fmt.Printf("  %s %2d. %-20s %s\n", mark, i+1, p.Name, p.APIBase)
				for _, m := range p.Models {
					mm := "  "
					if p.Name == cfg.ActiveProvider && m == cfg.ActiveModel {
						mm = " *"
					}
					fmt.Printf("    %s %s\n", mm, m)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

func providersAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new provider interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := agent.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := agent.LoadConfig(path)
			if err != nil {
				cfg = agent.DefaultConfig()
				cfg.Providers = nil
			}

			reader := bufio.NewReader(os.Stdin)
			var existingNames []string
			for _, pr := range cfg.Providers {
				existingNames = append(existingNames, pr.Name)
			}
			p, err := promptProvider(reader, len(cfg.Providers)+1, existingNames...)
			if err != nil {
				return err
			}
			cfg.Providers = append(cfg.Providers, p)
			if len(cfg.Providers) == 1 {
				cfg.ActiveProvider = p.Name
				if len(p.Models) > 0 {
					cfg.ActiveModel = p.Models[0]
				}
			}

			if err := agent.SaveConfig(path, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Provider '%s' added.\n", p.Name)
			return nil
		},
	}
}

func providersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name-or-number>",
		Short: "Delete a provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadProviders()
			if err != nil {
				return err
			}

			del := resolveProvider(cfg, args[0])
			if del == nil {
				return fmt.Errorf("provider '%s' not found", args[0])
			}

			var newProviders []agent.Provider
			for _, p := range cfg.Providers {
				if p.Name != del.Name {
					newProviders = append(newProviders, p)
				}
			}
			cfg.Providers = newProviders

			if del.Name == cfg.ActiveProvider {
				if len(cfg.Providers) > 0 {
					cfg.ActiveProvider = cfg.Providers[0].Name
					if len(cfg.Providers[0].Models) > 0 {
						cfg.ActiveModel = cfg.Providers[0].Models[0]
					}
				} else {
					cfg.ActiveProvider = ""
					cfg.ActiveModel = ""
				}
			}

			if err := agent.SaveConfig(path, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Provider '%s' deleted.\n", del.Name)
			return nil
		},
	}
}

func providersConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Interactive provider configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := agent.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := agent.LoadConfig(path)
			if err != nil {
				cfg = agent.DefaultConfig()
			}

			reader := bufio.NewReader(os.Stdin)

			for {
				fmt.Println("\n━━━ Provider Config ━━━")
				fmt.Printf("Active: %s / %s\n", cfg.ActiveProvider, cfg.ActiveModel)
				fmt.Println()
				for i, p := range cfg.Providers {
					mark := " "
					if p.Name == cfg.ActiveProvider {
						mark = ">"
					}
					fmt.Printf("  %s %2d. %-20s %s\n", mark, i+1, p.Name, p.APIBase)
				}
				fmt.Println()
				fmt.Println("  1) Add provider")
				fmt.Println("  2) Delete provider")
				fmt.Println("  3) Edit provider")
				fmt.Println("  4) Switch active provider")
				fmt.Println("  5) Save and quit")
				fmt.Print("Choice: ")

				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(choice)

				switch choice {
				case "5", "q":
					if err := agent.SaveConfig(path, cfg); err != nil {
						return fmt.Errorf("save config: %w", err)
					}
					fmt.Println("Config saved.")
					return nil

				case "1":
					fmt.Println()
					var existingNames []string
					for _, pr := range cfg.Providers {
						existingNames = append(existingNames, pr.Name)
					}
					p, err := promptProvider(reader, len(cfg.Providers)+1, existingNames...)
					if err != nil {
						return err
					}
					cfg.Providers = append(cfg.Providers, p)
					if len(cfg.Providers) == 1 {
						cfg.ActiveProvider = p.Name
						if len(p.Models) > 0 {
							cfg.ActiveModel = p.Models[0]
						}
					}

				case "2":
					if len(cfg.Providers) == 0 {
						fmt.Println("No providers configured.")
						continue
					}
					fmt.Print("Enter provider number or name to delete (q to cancel): ")
					target, _ := reader.ReadString('\n')
					target = strings.TrimSpace(target)
					if target == "q" {
						continue
					}
					del := resolveProvider(cfg, target)
					if del == nil {
						fmt.Println("Provider not found.")
						continue
					}
					var newProviders []agent.Provider
					for _, pr := range cfg.Providers {
						if pr.Name != del.Name {
							newProviders = append(newProviders, pr)
						}
					}
					cfg.Providers = newProviders
					if del.Name == cfg.ActiveProvider && len(cfg.Providers) > 0 {
						cfg.ActiveProvider = cfg.Providers[0].Name
						if len(cfg.Providers[0].Models) > 0 {
							cfg.ActiveModel = cfg.Providers[0].Models[0]
						}
					}
					fmt.Printf("Deleted '%s'.\n", del.Name)

				case "3":
					if len(cfg.Providers) == 0 {
						fmt.Println("No providers configured.")
						continue
					}
					fmt.Print("Enter provider number or name to edit (q to cancel): ")
					target, _ := reader.ReadString('\n')
					target = strings.TrimSpace(target)
					if target == "q" {
						continue
					}
					ed := resolveProvider(cfg, target)
					if ed == nil {
						fmt.Println("Provider not found.")
						continue
					}
					fmt.Printf("Editing '%s' (q to cancel, leave blank to keep value)\n", ed.Name)
					fmt.Printf("  Name [%s]: ", ed.Name)
					name, _ := reader.ReadString('\n')
					name = strings.TrimSpace(name)
					if name == "q" {
						continue
					}
					if name != "" {
						ed.Name = name
					}
					fmt.Printf("  API Base URL [%s]: ", ed.APIBase)
					base, _ := reader.ReadString('\n')
					base = strings.TrimSpace(base)
					if base == "q" {
						continue
					}
					if base != "" {
						ed.APIBase = base
					}
					fmt.Printf("  API Key [current: %s] (q to cancel): ", maskKey(ed.APIKey))
					key, _ := reader.ReadString('\n')
					key = strings.TrimSpace(key)
					if key == "q" {
						continue
					}
					if key != "" {
						ed.APIKey = key
					}
					fmt.Println("  Fetching models...")
					apiModels, err := agent.FetchModels(ed.APIBase, ed.APIKey)
					if err != nil {
						fmt.Printf("  Warning: could not fetch models (%v)\n", err)
						fmt.Printf("  Models (comma-separated, q to cancel) [%s]: ", strings.Join(ed.Models, ","))
						modelsStr, _ := reader.ReadString('\n')
						modelsStr = strings.TrimSpace(modelsStr)
						if modelsStr == "q" {
							continue
						}
						if modelsStr != "" {
							var ms []string
							for _, m := range strings.Split(modelsStr, ",") {
								m = strings.TrimSpace(m)
								if m != "" {
									ms = append(ms, m)
								}
							}
							if len(ms) > 0 {
								ed.Models = ms
							}
						}
					} else {
						for i, m := range apiModels {
							fmt.Printf("    %d. %s\n", i+1, m)
						}
						fmt.Printf("  Select models by number (comma-separated, Enter to keep, q to cancel): ")
						sel, _ := reader.ReadString('\n')
						sel = strings.TrimSpace(sel)
						if sel == "q" {
							continue
						}
						if sel != "" {
							var ms []string
							for _, s := range strings.Split(sel, ",") {
								s = strings.TrimSpace(s)
								idx, err := strconv.Atoi(s)
								if err == nil && idx >= 1 && idx <= len(apiModels) {
									ms = append(ms, apiModels[idx-1])
								}
							}
							if len(ms) > 0 {
								ed.Models = ms
							}
						}
					}
					fmt.Printf("Provider '%s' updated.\n", ed.Name)

				case "4":
					if len(cfg.Providers) == 0 {
						fmt.Println("No providers configured.")
						continue
					}
					fmt.Print("Enter provider number or name (q to cancel): ")
					target, _ := reader.ReadString('\n')
					target = strings.TrimSpace(target)
					if target == "q" {
						continue
					}
					p := resolveProvider(cfg, target)
					if p == nil {
						fmt.Println("Provider not found.")
						continue
					}
					cfg.ActiveProvider = p.Name
					if len(p.Models) > 0 {
						fmt.Println("Models:")
						for i, m := range p.Models {
							fmt.Printf("  %d. %s\n", i+1, m)
						}
						fmt.Printf("Select model [1] (q to cancel): ")
						sel, _ := reader.ReadString('\n')
						sel = strings.TrimSpace(sel)
						if sel == "q" {
							continue
						}
						if sel == "" {
							cfg.ActiveModel = p.Models[0]
						} else if idx, err := strconv.Atoi(sel); err == nil && idx >= 1 && idx <= len(p.Models) {
							cfg.ActiveModel = p.Models[idx-1]
						} else {
							cfg.ActiveModel = p.Models[0]
						}
					}
					fmt.Printf("Activated: %s / %s\n", cfg.ActiveProvider, cfg.ActiveModel)
				}
			}
		},
	}
}
