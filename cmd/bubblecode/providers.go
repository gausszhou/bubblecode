package bubblecode

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
)

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

			var existingNames []string
			for _, pr := range cfg.Providers {
				existingNames = append(existingNames, pr.Name)
			}
			p, err := promptProvider(len(cfg.Providers)+1, existingNames...)
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

func providerPicker(cfg *agent.Config, title string) *agent.Provider {
	opts := make([]huh.Option[string], len(cfg.Providers))
	for i, p := range cfg.Providers {
		opts[i] = huh.NewOption(p.Name, p.Name)
	}
	var name string
	huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&name).
		Run()
	if name == "" {
		return nil
	}
	return resolveProvider(cfg, name)
}

func providerConfigScreen(path string, cfg *agent.Config) error {
	for {
		fmt.Printf("\n  Active: %s / %s\n\n", cfg.ActiveProvider, cfg.ActiveModel)
		for i, p := range cfg.Providers {
			mark := " "
			if p.Name == cfg.ActiveProvider {
				mark = ">"
			}
			fmt.Printf("  %s %2d. %-20s %s\n", mark, i+1, p.Name, p.APIBase)
		}
		fmt.Println()

		var choice string
		opts := []huh.Option[string]{
			huh.NewOption("Add provider", "add"),
			huh.NewOption("Delete provider", "del"),
			huh.NewOption("Edit provider", "edit"),
			huh.NewOption("Switch active provider", "switch"),
			huh.NewOption("Save and quit", "quit"),
		}
		err := huh.NewSelect[string]().
			Title("Provider Config").
			Options(opts...).
			Value(&choice).
			Run()
		if err != nil {
			return err
		}

		switch choice {
		case "quit":
			if err := agent.SaveConfig(path, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("Config saved.")
			return nil

		case "add":
			var existingNames []string
			for _, pr := range cfg.Providers {
				existingNames = append(existingNames, pr.Name)
			}
			p, err := promptProvider(len(cfg.Providers)+1, existingNames...)
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

		case "del":
			if len(cfg.Providers) == 0 {
				fmt.Println("No providers configured.")
				continue
			}
			del := providerPicker(cfg, "Select provider to delete")
			if del == nil {
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

		case "edit":
			if len(cfg.Providers) == 0 {
				fmt.Println("No providers configured.")
				continue
			}
			ed := providerPicker(cfg, "Select provider to edit")
			if ed == nil {
				continue
			}
			huh.NewInput().
				Title("Name").
				Value(&ed.Name).
				Run()
			huh.NewInput().
				Title("API Base URL").
				Value(&ed.APIBase).
				Run()
			huh.NewInput().
				Title("API Key").
				EchoMode(huh.EchoModePassword).
				Value(&ed.APIKey).
				Run()
			fmt.Println("  Fetching models...")
			apiModels, err := agent.FetchModels(ed.APIBase, ed.APIKey)
			if err != nil {
				fmt.Printf("  Warning: could not fetch models (%v)\n", err)
				var modelsStr string
				huh.NewInput().
					Title("Models (comma-separated)").
					Value(&modelsStr).
					Run()
				if modelsStr != "" {
					ed.Models = splitComma(modelsStr)
				}
			} else {
				modelOpts := make([]huh.Option[string], len(apiModels))
				for i, m := range apiModels {
					modelOpts[i] = huh.NewOption(m, m)
				}
				var selected []string
				huh.NewMultiSelect[string]().
					Title("Select models (space to toggle, enter to confirm)").
					Options(modelOpts...).
					Value(&selected).
					Run()
				if len(selected) > 0 {
					ed.Models = selected
				}
			}
			fmt.Printf("Provider '%s' updated.\n", ed.Name)

		case "switch":
			if len(cfg.Providers) == 0 {
				fmt.Println("No providers configured.")
				continue
			}
			p := providerPicker(cfg, "Select provider to activate")
			if p == nil {
				continue
			}
			cfg.ActiveProvider = p.Name
			if len(p.Models) > 0 {
				if len(p.Models) == 1 {
					cfg.ActiveModel = p.Models[0]
				} else {
					modelOpts := make([]huh.Option[string], len(p.Models))
					for i, m := range p.Models {
						modelOpts[i] = huh.NewOption(m, m)
					}
					var model string
					huh.NewSelect[string]().
						Title("Select model").
						Options(modelOpts...).
						Value(&model).
						Run()
					if model != "" {
						cfg.ActiveModel = model
					}
				}
			}
			fmt.Printf("Activated: %s / %s\n", cfg.ActiveProvider, cfg.ActiveModel)
		}
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
			return providerConfigScreen(path, cfg)
		},
	}
}
